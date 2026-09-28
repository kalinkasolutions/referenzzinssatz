package consolelogsink

import (
	"fmt"
	"time"

	"github.com/kalinkasolutions/referenzzinssatz/logger"
)

type ConsoleSink struct {
}

func NewConsoleSink() *ConsoleSink {
	return &ConsoleSink{}
}

func (l *ConsoleSink) Name() string {
	return "consolelogger"
}

func (l *ConsoleSink) Log(message string, level int, now time.Time) {
	fmt.Println(now.Format(time.ANSIC) + "\t" + logger.LogLevelText(level) + "\t\t" + message)
}
