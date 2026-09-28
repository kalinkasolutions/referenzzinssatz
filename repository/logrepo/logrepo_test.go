package logrepo_test

import (
	"testing"
	"time"

	"github.com/go-playground/assert/v2"
	"github.com/kalinkasolutions/referenzzinssatz/config"
	"github.com/kalinkasolutions/referenzzinssatz/datalayer"
	"github.com/kalinkasolutions/referenzzinssatz/mocks"
	"github.com/kalinkasolutions/referenzzinssatz/repository/logrepo"
)

func TestInsertAndGetAll(t *testing.T) {
	db := datalayer.NewDb(mocks.NewLoggerMock(), config.Config{
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
	db := datalayer.NewDb(mocks.NewLoggerMock(), config.Config{
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
