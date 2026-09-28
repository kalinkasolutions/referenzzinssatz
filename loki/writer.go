// Package loki ships log lines to Grafana Loki. The Writer is meant to sit
// behind an slog.JSONHandler, which writes exactly one line per record.
package loki

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kalinkasolutions/referenzzinssatz/config"
)

const (
	serviceName   = "referenzzinssatz"
	pushPath      = "/loki/api/v1/push"
	flushInterval = 2 * time.Second
	maxBatchSize  = 500
	// While Loki is unreachable, lines beyond this are dropped instead of piling up in memory.
	maxPending  = 10000
	pushTimeout = 10 * time.Second
)

type Writer struct {
	config   config.LokiConfig
	pushUrl  string
	labels   map[string]string
	client   *http.Client
	flushNow chan struct{}

	mutex   sync.Mutex
	pending []entry
	dropped int
}

type entry struct {
	timestamp time.Time
	line      string
}

type pushRequest struct {
	Streams []stream `json:"streams"`
}

type stream struct {
	Stream map[string]string `json:"stream"`
	Values [][2]string       `json:"values"`
}

// NewWriter starts a background loop that pushes collected lines to Loki.
func NewWriter(lokiConfig config.LokiConfig) *Writer {
	labels := map[string]string{"service_name": serviceName}
	maps.Copy(labels, lokiConfig.Labels)

	writer := &Writer{
		config:   lokiConfig,
		pushUrl:  PushUrl(lokiConfig.Url),
		labels:   labels,
		client:   &http.Client{Timeout: pushTimeout},
		flushNow: make(chan struct{}, 1),
	}
	go writer.flushPeriodically()
	return writer
}

// Write never blocks on the network, so logging stays cheap even when Loki is slow.
func (w *Writer) Write(line []byte) (int, error) {
	w.mutex.Lock()
	if len(w.pending) < maxPending {
		w.pending = append(w.pending, entry{timestamp: time.Now(), line: strings.TrimSuffix(string(line), "\n")})
	} else {
		w.dropped++
	}
	batchFull := len(w.pending) >= maxBatchSize
	w.mutex.Unlock()

	if batchFull {
		select {
		case w.flushNow <- struct{}{}:
		default:
		}
	}
	return len(line), nil
}

// Flush pushes everything collected so far. Failures go to stderr, since logging them could recurse.
func (w *Writer) Flush() {
	w.mutex.Lock()
	batch, dropped := w.pending, w.dropped
	w.pending, w.dropped = nil, 0
	w.mutex.Unlock()

	if dropped > 0 {
		fmt.Fprintf(os.Stderr, "Dropped %d log lines because Loki could not keep up\n", dropped)
	}
	if len(batch) == 0 {
		return
	}
	if err := w.push(batch); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to push %d log lines to Loki: %v\n", len(batch), err)
	}
}

func (w *Writer) flushPeriodically() {
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
		case <-w.flushNow:
		}
		w.Flush()
	}
}

func (w *Writer) push(batch []entry) error {
	body, err := json.Marshal(buildPushRequest(w.labels, batch))
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), pushTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, w.pushUrl, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	if w.config.Username != "" || w.config.Password != "" {
		request.SetBasicAuth(w.config.Username, w.config.Password)
	}
	if w.config.TenantId != "" {
		request.Header.Set("X-Scope-OrgID", w.config.TenantId)
	}

	response, err := w.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode >= http.StatusMultipleChoices {
		detail, _ := io.ReadAll(io.LimitReader(response.Body, 512))
		return fmt.Errorf("loki answered %s: %s", response.Status, strings.TrimSpace(string(detail)))
	}
	return nil
}

// PushUrl accepts either the full push endpoint or just the Loki address,
// e.g. http://loki.example.ch:3100, and adds the push path to the latter.
func PushUrl(configured string) string {
	parsed, err := url.Parse(configured)
	if err != nil || (parsed.Path != "" && parsed.Path != "/") {
		return configured
	}
	parsed.Path = pushPath
	return parsed.String()
}

// buildPushRequest puts all lines in one stream; Loki wants timestamps as nanosecond strings.
func buildPushRequest(labels map[string]string, batch []entry) pushRequest {
	values := make([][2]string, 0, len(batch))
	for _, entry := range batch {
		values = append(values, [2]string{strconv.FormatInt(entry.timestamp.UnixNano(), 10), entry.line})
	}
	return pushRequest{Streams: []stream{{Stream: labels, Values: values}}}
}
