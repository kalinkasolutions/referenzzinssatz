package dblog

import (
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/go-playground/assert/v2"
	"github.com/kalinkasolutions/referenzzinssatz/mocks"
)

func TestStoresOnlyRecordsAtOrAboveLevel(t *testing.T) {
	repo := &mocks.LogRepositoryMock{}
	logger := slog.New(NewHandler(repo, slog.LevelWarn))

	logger.Info("ignored")
	logger.Warn("stored")

	assert.Equal(t, 1, len(repo.Inserted))
	assert.Equal(t, "stored", repo.Inserted[0].Message)
	assert.Equal(t, int(slog.LevelWarn), repo.Inserted[0].Level)
	assert.Equal(t, "", repo.Inserted[0].Attributes)
}

func TestStoresFieldsAsJson(t *testing.T) {
	repo := &mocks.LogRepositoryMock{}
	logger := slog.New(NewHandler(repo, slog.LevelInfo))

	logger.With("component", "mail").Error("Failed to send mail",
		"to", "a@example.ch",
		"error", errors.New("timeout"),
		"attempt", 2,
		"duration", 1500*time.Millisecond,
	)

	assert.Equal(t, `{"attempt":2,"component":"mail","duration":"1.5s","error":"timeout","to":"a@example.ch"}`, repo.Inserted[0].Attributes)
}

func TestFlattensGroups(t *testing.T) {
	repo := &mocks.LogRepositoryMock{}
	logger := slog.New(NewHandler(repo, slog.LevelInfo))

	logger.WithGroup("request").With("method", "GET").Info("handled", slog.Group("response", "status", 200))

	assert.Equal(t, `{"request.method":"GET","request.response.status":200}`, repo.Inserted[0].Attributes)
}

func TestWithAttrsDoesNotLeakBetweenLoggers(t *testing.T) {
	repo := &mocks.LogRepositoryMock{}
	base := slog.New(NewHandler(repo, slog.LevelInfo))

	base.With("a", 1).Info("first")
	base.With("b", 2).Info("second")

	assert.Equal(t, `{"a":1}`, repo.Inserted[0].Attributes)
	assert.Equal(t, `{"b":2}`, repo.Inserted[1].Attributes)
}
