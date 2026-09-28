package loki

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-playground/assert/v2"
	"github.com/kalinkasolutions/referenzzinssatz/config"
)

type receivedPush struct {
	request  pushRequest
	username string
	password string
	tenant   string
}

func newLokiStub(t *testing.T, status int) (*httptest.Server, chan receivedPush) {
	received := make(chan receivedPush, 10)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var push receivedPush
		if err := json.NewDecoder(r.Body).Decode(&push.request); err != nil {
			t.Errorf("invalid push body: %v", err)
		}
		push.username, push.password, _ = r.BasicAuth()
		push.tenant = r.Header.Get("X-Scope-OrgID")
		received <- push
		w.WriteHeader(status)
	}))
	t.Cleanup(server.Close)
	return server, received
}

func TestPushesSlogRecordsWithLabelsAndAuth(t *testing.T) {
	server, received := newLokiStub(t, http.StatusNoContent)
	writer := NewWriter(config.LokiConfig{
		Url:      server.URL,
		Username: "user",
		Password: "secret",
		TenantId: "tenant-1",
		Labels:   map[string]string{"environment": "test"},
	})
	logger := slog.New(slog.NewJSONHandler(writer, nil))

	logger.Info("Subscription confirmed", "subscriberId", "s-1")
	logger.Warn("reCAPTCHA score below minimum", "score", 0.3)
	writer.Flush()

	push := <-received
	assert.Equal(t, "user", push.username)
	assert.Equal(t, "secret", push.password)
	assert.Equal(t, "tenant-1", push.tenant)
	assert.Equal(t, 1, len(push.request.Streams))
	assert.Equal(t, map[string]string{"service_name": "referenzzinssatz", "environment": "test"}, push.request.Streams[0].Stream)

	values := push.request.Streams[0].Values
	assert.Equal(t, 2, len(values))
	assert.Equal(t, true, strings.Contains(values[0][1], `"msg":"Subscription confirmed"`))
	assert.Equal(t, true, strings.Contains(values[0][1], `"subscriberId":"s-1"`))
	assert.Equal(t, false, strings.HasSuffix(values[0][1], "\n"))
	assert.Equal(t, true, strings.Contains(values[1][1], `"level":"WARN"`))
}

func TestFlushSendsEachLineOnce(t *testing.T) {
	server, received := newLokiStub(t, http.StatusNoContent)
	writer := NewWriter(config.LokiConfig{Url: server.URL})

	writer.Write([]byte("first\n"))
	writer.Flush()
	writer.Flush()

	assert.Equal(t, 1, len((<-received).request.Streams[0].Values))
	select {
	case <-received:
		t.Fatal("an empty flush should not push")
	case <-time.After(100 * time.Millisecond):
	}
}

func TestFullBatchIsPushedWithoutWaitingForTicker(t *testing.T) {
	server, received := newLokiStub(t, http.StatusNoContent)
	writer := NewWriter(config.LokiConfig{Url: server.URL})

	for range maxBatchSize {
		writer.Write([]byte("line\n"))
	}

	select {
	case push := <-received:
		assert.Equal(t, maxBatchSize, len(push.request.Streams[0].Values))
	case <-time.After(flushInterval / 2):
		t.Fatal("full batch was not pushed early")
	}
}

func TestPushReportsLokiErrors(t *testing.T) {
	server, _ := newLokiStub(t, http.StatusBadRequest)
	writer := &Writer{config: config.LokiConfig{Url: server.URL}, client: http.DefaultClient}

	err := writer.push([]entry{{timestamp: time.Now(), line: "x"}})

	assert.NotEqual(t, nil, err)
}

func TestConfiguredLabelsCanOverrideServiceName(t *testing.T) {
	writer := NewWriter(config.LokiConfig{Url: "http://unused", Labels: map[string]string{"service_name": "staging"}})

	assert.Equal(t, "staging", writer.labels["service_name"])
}

func TestBuildPushRequestUsesNanosecondTimestamps(t *testing.T) {
	timestamp := time.Date(2025, 9, 2, 8, 0, 0, 123, time.UTC)

	request := buildPushRequest(map[string]string{"service_name": "x"}, []entry{{timestamp: timestamp, line: "hello"}})

	assert.Equal(t, [2]string{"1756800000000000123", "hello"}, request.Streams[0].Values[0])
}
