package mocks

import (
	"time"

	"github.com/kalinkasolutions/referenzzinssatz/repository/logrepo"
)

type LogRepositoryMock struct {
	Inserted []logrepo.Log
}

func (l *LogRepositoryMock) Insert(logEntry logrepo.Log) {
	l.Inserted = append(l.Inserted, logEntry)
}

func (l *LogRepositoryMock) GetAll() []logrepo.Log {
	return l.Inserted
}

func (l *LogRepositoryMock) DeleteOlderThan(cutoff time.Time) {
}
