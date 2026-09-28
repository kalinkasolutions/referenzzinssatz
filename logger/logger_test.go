package logger

import (
	"testing"
	"time"

	"github.com/go-playground/assert/v2"
)

func TestAddSinksInConstructor(t *testing.T) {
	sink_1 := NewTestSink_1("sink_1")
	sink_2 := NewTestSink_1("sink_2")
	logger := NewLogger(sink_1, sink_2)
	logger.Debug("log")
	assert.Equal(t, 1, sink_1.called)
	assert.Equal(t, 1, sink_2.called)
}

func TestLogLevel(t *testing.T) {
	sink_1 := NewTestSink_1("sink_1")
	logger := NewLogger(sink_1)
	logger.Debug("log")
	assert.Equal(t, 0, sink_1.level)
	logger.Info("log")
	assert.Equal(t, 1, sink_1.level)
	logger.Warning("log")
	assert.Equal(t, 2, sink_1.level)
	logger.Error("log")
	assert.Equal(t, 3, sink_1.level)
}

func TestAddSink(t *testing.T) {
	sink_1 := NewTestSink_1("sink_1")
	logger := NewLogger()
	logger.AddSink(sink_1)
	logger.Debug("log")
	assert.Equal(t, 1, sink_1.LogCalledCount())
}

func TestRemoveSink(t *testing.T) {
	sink_1 := NewTestSink_1("sink_1")
	logger := NewLogger()
	logger.AddSink(sink_1)

	logger.Debug("log")

	assert.Equal(t, 1, sink_1.LogCalledCount())

	logger.RemoveSink(sink_1)
	logger.Debug("log")

	assert.Equal(t, 1, sink_1.LogCalledCount())
}

func TestLogParams(t *testing.T) {
	sink_1 := NewTestSink_1("sink_1")
	logger := NewLogger(sink_1)

	logger.Debug("log %s %s", "a", "b")

	assert.Equal(t, "log a b", sink_1.message)
	assert.NotEqual(t, nil, sink_1.now)
}

func TestSinkWithSameNameIsOverwritten(t *testing.T) {
	sink_1 := NewTestSink_1("sink_1")
	sink_2 := NewTestSink_1("sink_1")
	logger := NewLogger(sink_1)

	logger.Debug("log")

	assert.Equal(t, 1, sink_1.LogCalledCount())

	logger.AddSink(sink_2)
	logger.Debug("log")

	assert.Equal(t, 1, sink_1.LogCalledCount())
	assert.Equal(t, 1, sink_2.LogCalledCount())
	assert.NotEqual(t, nil, sink_1.now)
}

type TestSink struct {
	name    string
	called  int
	message string
	level   int
	now     time.Time
}

func NewTestSink_1(name string) *TestSink {
	return &TestSink{name: name}
}

func (t *TestSink) Name() string {
	return t.name
}

func (t *TestSink) Log(message string, level int, now time.Time) {
	t.message = message
	t.level = level
	t.now = now
	t.called += 1
}

func (t *TestSink) LogCalledCount() int {
	return t.called
}
func (t *TestSink) Message() string {
	return t.message
}
func (t *TestSink) Level() int {
	return t.level
}
func (t *TestSink) Now() time.Time {
	return t.now
}
