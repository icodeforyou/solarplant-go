package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"slices"

	"github.com/angas/solarplant-go/database"
)

type LogAttrFormat string

const (
	LogAttrFormatText LogAttrFormat = "TEXT"
	LogAttrFormatJSON LogAttrFormat = "JSON"
)

type SQLiteHandler struct {
	db       *database.Database
	minLevel slog.Level
	format   LogAttrFormat
	attrs    []slog.Attr
	groups   []string
}

func NewSQLiteHandler(db *database.Database, minLevel slog.Level, format LogAttrFormat) *SQLiteHandler {
	return &SQLiteHandler{
		db:       db,
		minLevel: minLevel,
		format:   format,
		attrs:    []slog.Attr{},
		groups:   []string{},
	}
}

func (h *SQLiteHandler) Handle(ctx context.Context, r slog.Record) error {
	if r.Level < h.minLevel {
		return nil
	}

	merged := make([]slog.Attr, 0, 20)
	merged = append(merged, h.attrs...)
	r.Attrs(func(a slog.Attr) bool {
		merged = append(merged, a)
		return true
	})

	attrsStr := ""
	if strings.EqualFold(string(h.format), "text") {
		var b strings.Builder
		for i, a := range merged {
			if i > 0 {
				b.WriteString("; ")
			}
			b.WriteString(a.Key)
			b.WriteString("=")
			b.WriteString(strings.ReplaceAll(strings.ReplaceAll(a.Value.String(), "=", "\\="), ";", "\\;"))
		}
		attrsStr = b.String()
	} else {
		var attrs []map[string]string
		for _, a := range merged {
			attrs = append(attrs, map[string]string{a.Key: a.Value.String()})
		}
		if len(attrs) > 0 {
			jsonBytes, err := json.Marshal(attrs)
			if err != nil {
				attrsStr = fmt.Sprintf(`{"error": "%v"}`, err)
			} else {
				attrsStr = string(jsonBytes)
			}
		}
	}

	return h.db.SaveLogEntry(ctx, database.LogEntryRow{
		Timestamp: time.Now(),
		Level:     int(r.Level),
		Message:   r.Message,
		Attrs:     attrsStr,
	})
}

func (h *SQLiteHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clone := *h
	clone.attrs = slices.Concat(h.attrs, attrs)
	return &clone
}

func (h *SQLiteHandler) WithGroup(name string) slog.Handler {
	clone := *h
	clone.groups = append(slices.Clone(h.groups), name)
	return &clone
}

func (h *SQLiteHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.minLevel
}
