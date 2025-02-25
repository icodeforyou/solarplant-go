package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

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
}

func NewSQLiteHandler(db *database.Database, minLevel slog.Level, format LogAttrFormat) *SQLiteHandler {
	return &SQLiteHandler{db: db, minLevel: minLevel, format: format}
}

func (h *SQLiteHandler) Handle(ctx context.Context, r slog.Record) error {
	if r.Level < h.minLevel {
		return nil
	}

	attrsStr := ""
	if strings.EqualFold(string(h.format), "text") {
		var b strings.Builder
		r.Attrs(func(a slog.Attr) bool {
			if b.Len() > 0 {
				b.WriteString("; ")
			}
			b.WriteString(a.Key)
			b.WriteString("=")
			b.WriteString(strings.ReplaceAll(strings.ReplaceAll(a.Value.String(), "=", "\\="), ";", "\\;"))
			return true
		})
		attrsStr = b.String()
	} else {
		var attrs []map[string]string
		r.Attrs(func(a slog.Attr) bool {
			attrs = append(attrs, map[string]string{a.Key: a.Value.String()})
			return true
		})
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
	return h
}

func (h *SQLiteHandler) WithGroup(name string) slog.Handler {
	return h
}

func (h *SQLiteHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.minLevel
}
