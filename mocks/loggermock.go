package mocks

import "github.com/kalinkasolutions/referenzzinssatz/logger"

type LoggerMock struct {
}

func NewLoggerMock() *LoggerMock {
	return &LoggerMock{}
}

func (l *LoggerMock) Debug(message string, args ...any) {
}

func (l *LoggerMock) Info(message string, args ...any) {
}

func (l *LoggerMock) Warning(message string, args ...any) {
}

func (l *LoggerMock) Error(message string, args ...any) {
}

func (l *LoggerMock) AddSink(sink logger.ISink) {
}

func (l *LoggerMock) RemoveSink(sink logger.ISink) {
}
