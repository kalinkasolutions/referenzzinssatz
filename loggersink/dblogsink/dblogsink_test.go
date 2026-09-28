package dblogsink

import (
	"testing"
	"time"

	"github.com/go-playground/assert/v2"
	"github.com/kalinkasolutions/referenzzinssatz/mocks"
)

func TestDbLoggerSink(t *testing.T) {
	message := "message"
	level := 1
	now := time.Now()
	repoMock := mocks.NewLogRepositoryMock()
	sink := NewDbSink(repoMock)

	sink.Log(message, level, now)

	loggedMessage, loggedLevel, loggedNow := repoMock.InsertArguments()

	assert.Equal(t, message, loggedMessage)
	assert.Equal(t, level, loggedLevel)
	assert.Equal(t, now.Format(time.RFC3339), loggedNow)

}
