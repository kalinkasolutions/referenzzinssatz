package logrepo

import (
	"database/sql"
	"log"
	"time"

	"github.com/google/uuid"
)

type Log struct {
	Id        string
	CreatedAt string
	Level     int
	Message   string
}

type ILogRepository interface {
	Insert(logEntry Log)
	GetAll() []Log
	DeleteOlderThan(cutoff time.Time)
}

// LogRepository reports its own failures with the standard logger, since it
// backs the application logger and would otherwise log into itself.
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
		(Id, CreatedAt, LogLevel, Message)
		VALUES (?, ?, ?, ?)`, id, logEntry.CreatedAt, logEntry.Level, logEntry.Message)
	if err != nil {
		log.Printf("Failed to insert log: %v", err)
	}
}

func (l *LogRepository) GetAll() []Log {
	var logs []Log

	rows, err := l.db.Query("SELECT Id, CreatedAt, LogLevel, Message FROM Logs")
	if err != nil {
		log.Printf("Failed to get logs: %v", err)
		return logs
	}
	defer rows.Close()

	for rows.Next() {
		var logEntry Log
		err := rows.Scan(&logEntry.Id, &logEntry.CreatedAt, &logEntry.Level, &logEntry.Message)
		if err != nil {
			log.Printf("Failed to read log row: %v", err)
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
		log.Printf("Failed to delete old logs: %v", err)
	}
}
