package mocks

import (
	"time"

	"github.com/kalinkasolutions/referenzzinssatz/repository/logrepo"
)

type LogRepositoryMock struct {
	message string
	level   int
	now     string
}

func NewLogRepositoryMock() *LogRepositoryMock {
	return &LogRepositoryMock{}
}

func (l *LogRepositoryMock) Insert(logEntry logrepo.Log) {
	l.message = logEntry.Message
	l.level = logEntry.Level
	l.now = logEntry.CreatedAt
}

func (l *LogRepositoryMock) InsertArguments() (string, int, string) {
	return l.message, l.level, l.now
}

func (l *LogRepositoryMock) GetAll() []logrepo.Log {
	return []logrepo.Log{}
}

func (l *LogRepositoryMock) DeleteOlderThan(cutoff time.Time) {
}
