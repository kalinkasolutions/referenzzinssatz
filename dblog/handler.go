// Package dblog is a slog.Handler that stores log records in the Logs table.
package dblog

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/kalinkasolutions/referenzzinssatz/repository/logrepo"
)

type Handler struct {
	repo  logrepo.ILogRepository
	level slog.Leveler
	// attrs come from Logger.With and already carry their group prefix.
	attrs       []slog.Attr
	groupPrefix string
}

func NewHandler(repo logrepo.ILogRepository, level slog.Leveler) *Handler {
	return &Handler{repo: repo, level: level}
}

func (h *Handler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

func (h *Handler) Handle(_ context.Context, record slog.Record) error {
	fields := make(map[string]any)
	for _, attr := range h.attrs {
		addField(fields, "", attr)
	}
	record.Attrs(func(attr slog.Attr) bool {
		addField(fields, h.groupPrefix, attr)
		return true
	})

	attributes := ""
	if len(fields) > 0 {
		encoded, err := json.Marshal(fields)
		if err != nil {
			return err
		}
		attributes = string(encoded)
	}

	h.repo.Insert(logrepo.Log{
		CreatedAt:  record.Time.Format(time.RFC3339),
		Level:      int(record.Level),
		Message:    record.Message,
		Attributes: attributes,
	})
	return nil
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := *h
	next.attrs = append([]slog.Attr{}, h.attrs...)
	for _, attr := range attrs {
		next.attrs = append(next.attrs, slog.Attr{Key: h.groupPrefix + attr.Key, Value: attr.Value})
	}
	return &next
}

func (h *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	next := *h
	next.groupPrefix = h.groupPrefix + name + "."
	return &next
}

// addField flattens groups into dotted keys, e.g. "request.path".
func addField(fields map[string]any, prefix string, attr slog.Attr) {
	value := attr.Value.Resolve()
	if attr.Key == "" && value.Kind() != slog.KindGroup {
		return
	}

	key := prefix + attr.Key
	if value.Kind() == slog.KindGroup {
		groupPrefix := prefix
		if attr.Key != "" {
			groupPrefix = key + "."
		}
		for _, groupAttr := range value.Group() {
			addField(fields, groupPrefix, groupAttr)
		}
		return
	}
	fields[key] = jsonValue(value)
}

func jsonValue(value slog.Value) any {
	switch value.Kind() {
	case slog.KindTime:
		return value.Time().Format(time.RFC3339)
	case slog.KindDuration:
		return value.Duration().String()
	case slog.KindAny:
		// Errors and arbitrary types would otherwise marshal as {} or fail.
		if err, ok := value.Any().(error); ok {
			return err.Error()
		}
		return fmt.Sprint(value.Any())
	default:
		return value.Any()
	}
}
