package logrepo

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
)

type Log struct {
	Id        string
	CreatedAt string
	Level     int
	Message   string
	// Attributes holds the record's key/value fields as a JSON object, or "" if there are none.
	Attributes string
}

type ILogRepository interface {
	Insert(logEntry Log)
	GetAll() []Log
	DeleteOlderThan(cutoff time.Time)
}

// LogRepository reports its own failures straight to stderr: it backs the
// application logger, so logging them there could recurse.
type LogRepository struct {
	db *sql.DB
}

func NewLogRepository(db *sql.DB) *LogRepository {
	return &LogRepository{
		db: db,
	}
}

func (l *LogRepository) Insert(logEntry Log) {
	id := uuid.New().String()
	_, err := l.db.Exec(`
	INSERT INTO Logs
		(Id, CreatedAt, LogLevel, Message, Attributes)
		VALUES (?, ?, ?, ?, ?)`, id, logEntry.CreatedAt, logEntry.Level, logEntry.Message, logEntry.Attributes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to insert log: %v\n", err)
	}
}

func (l *LogRepository) GetAll() []Log {
	var logs []Log

	rows, err := l.db.Query("SELECT Id, CreatedAt, LogLevel, Message, Attributes FROM Logs")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get logs: %v\n", err)
		return logs
	}
	defer rows.Close()

	for rows.Next() {
		var logEntry Log
		err := rows.Scan(&logEntry.Id, &logEntry.CreatedAt, &logEntry.Level, &logEntry.Message, &logEntry.Attributes)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read log row: %v\n", err)
			continue
		}
		logs = append(logs, logEntry)
	}
	return logs
}

func (l *LogRepository) DeleteOlderThan(cutoff time.Time) {
	// datetime() normalizes the stored RFC3339 timestamps, whatever their offset.
	_, err := l.db.Exec("DELETE FROM Logs WHERE datetime(CreatedAt) < datetime(?)", cutoff.Format(time.RFC3339))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to delete old logs: %v\n", err)
	}
}
