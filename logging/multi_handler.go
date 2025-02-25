package logging

import (
	"context"
	"log/slog"
	"sync"
)

type MultiHandler struct {
	mu       *sync.Mutex
	handlers []slog.Handler
}

func NewMultiHandler(handlers ...slog.Handler) *MultiHandler {
	h := &MultiHandler{handlers: handlers, mu: &sync.Mutex{}}
	return h
}

func (h *MultiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}

func (h *MultiHandler) Handle(ctx context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, destHandler := range h.handlers {
		err := destHandler.Handle(ctx, r)
		if err != nil {
			return err
		}
	}
	return nil
}

func (h *MultiHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	h2 := *h
	h2.handlers = make([]slog.Handler, len(h.handlers))
	for i, h := range h.handlers {
		h2.handlers[i] = h.WithGroup(name)
	}
	return &h2
}

func (h *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	h2 := *h
	h2.handlers = make([]slog.Handler, len(h.handlers))
	for i, h := range h.handlers {
		h2.handlers[i] = h.WithAttrs(attrs)
	}
	return &h2
}
