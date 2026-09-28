package logrepo_test

import (
	"log/slog"
	"testing"
	"time"

	"github.com/go-playground/assert/v2"
	"github.com/kalinkasolutions/referenzzinssatz/config"
	"github.com/kalinkasolutions/referenzzinssatz/datalayer"
	"github.com/kalinkasolutions/referenzzinssatz/repository/logrepo"
)

func TestInsertAndGetAll(t *testing.T) {
	db := datalayer.NewDb(slog.New(slog.DiscardHandler), config.Config{
		DatabaseName: "file::memory:?cache=shared",
	})
	defer db.Close()
	repo := logrepo.NewLogRepository(db)

	repo.Insert(logrepo.Log{CreatedAt: time.Now().Format(time.RFC3339), Level: 1, Message: "message"})
	logs := repo.GetAll()

	assert.Equal(t, 1, len(logs))
	assert.NotEqual(t, "", logs[0].Id)
	assert.Equal(t, 1, logs[0].Level)
	assert.Equal(t, "message", logs[0].Message)
}

func TestDeleteOlderThan(t *testing.T) {
	db := datalayer.NewDb(slog.New(slog.DiscardHandler), config.Config{
		DatabaseName: "file::memory:?cache=shared",
	})
	defer db.Close()
	repo := logrepo.NewLogRepository(db)
	now := time.Date(2025, 9, 28, 12, 0, 0, 0, time.FixedZone("CEST", 2*60*60))

	repo.Insert(logrepo.Log{CreatedAt: now.AddDate(0, 0, -100).Format(time.RFC3339), Message: "old"})
	repo.Insert(logrepo.Log{CreatedAt: now.Format(time.RFC3339), Message: "recent"})

	repo.DeleteOlderThan(now.AddDate(0, 0, -90))
	logs := repo.GetAll()

	assert.Equal(t, 1, len(logs))
	assert.Equal(t, "recent", logs[0].Message)
}

func TestMigrationMapsOldLevelsToSlog(t *testing.T) {
	db := datalayer.NewDb(slog.New(slog.DiscardHandler), config.Config{
		DatabaseName: "file::memory:?cache=shared",
	})
	defer db.Close()
	// Recreate a version 3 database holding one entry per old level (debug, info, warning, error).
	_, err := db.Exec(`ALTER TABLE Logs DROP COLUMN Attributes; PRAGMA user_version = 3;
		INSERT INTO Logs VALUES ('0', '2025-01-01T00:00:00Z', 0, 'debug'), ('1', '2025-01-01T00:00:00Z', 1, 'info'),
		                        ('2', '2025-01-01T00:00:00Z', 2, 'warning'), ('3', '2025-01-01T00:00:00Z', 3, 'error')`)
	assert.Equal(t, nil, err)

	datalayer.NewDb(slog.New(slog.DiscardHandler), config.Config{DatabaseName: "file::memory:?cache=shared"}).Close()
	levels := map[string]int{}
	for _, entry := range logrepo.NewLogRepository(db).GetAll() {
		levels[entry.Message] = entry.Level
	}

	assert.Equal(t, map[string]int{"debug": int(slog.LevelDebug), "info": int(slog.LevelInfo), "warning": int(slog.LevelWarn), "error": int(slog.LevelError)}, levels)
}
