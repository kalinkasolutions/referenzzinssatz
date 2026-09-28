package consolelogsink

import (
	"io"
	"os"
	"testing"
	"time"

	"github.com/go-playground/assert/v2"
)

func TestConsoelSink(t *testing.T) {
	rescueStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	consoleSink := NewConsoleSink()
	now := time.Now()

	consoleSink.Log("log", 1, now)

	w.Close()
	out, _ := io.ReadAll(r)
	os.Stdout = rescueStdout

	assert.Equal(t, string(out), now.Format(time.ANSIC)+"\tINFO\t\tlog\n")
}
