package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

var (
	instance *slog.Logger
	buffer   *RingBuffer
)

// Init creates the global logger with JSON output to stdout + ring buffer.
// level: "debug", "info", "warn", "error" (default: "info")
func Init(level string) {
	var logLevel slog.Level
	switch strings.ToLower(level) {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	buffer = NewRingBuffer(500)

	stdoutHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	})

	multi := &multiHandler{
		handlers: []slog.Handler{stdoutHandler, buffer},
	}

	instance = slog.New(multi)
	slog.SetDefault(instance)
}

// Get returns the global logger. Falls back to default if Init wasn't called.
func Get() *slog.Logger {
	if instance == nil {
		return slog.Default()
	}
	return instance
}

// Buffer returns the ring buffer for reading log entries.
func Buffer() *RingBuffer {
	return buffer
}

// multiHandler fans out log records to multiple handlers.
type multiHandler struct {
	handlers []slog.Handler
}

func (m *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range m.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (m *multiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, h := range m.handlers {
		if err := h.Handle(ctx, r); err != nil {
			return err
		}
	}
	return nil
}

func (m *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		handlers[i] = h.WithAttrs(attrs)
	}
	return &multiHandler{handlers: handlers}
}

func (m *multiHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		handlers[i] = h.WithGroup(name)
	}
	return &multiHandler{handlers: handlers}
}
