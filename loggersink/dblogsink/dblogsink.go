package dblogsink

import (
	"time"

	"github.com/kalinkasolutions/referenzzinssatz/repository/logrepo"
)

type DbSink struct {
	logRepository logrepo.ILogRepository
}

func NewDbSink(logRepo logrepo.ILogRepository) *DbSink {
	return &DbSink{
		logRepository: logRepo,
	}
}

func (d DbSink) Name() string {
	return "dblogger"
}

func (d *DbSink) Log(message string, level int, now time.Time) {
	log := logrepo.Log{
		Message:   message,
		CreatedAt: now.Format(time.RFC3339),
		Level:     level,
	}
	d.logRepository.Insert(log)
}
