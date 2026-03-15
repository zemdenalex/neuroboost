package logger

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"
)

// LogEntry represents a single log entry stored in the ring buffer.
type LogEntry struct {
	Timestamp time.Time         `json:"timestamp"`
	Level     string            `json:"level"`
	Message   string            `json:"message"`
	Fields    map[string]string `json:"fields,omitempty"`
}

// RingBuffer is a fixed-size circular buffer that implements slog.Handler.
type RingBuffer struct {
	mu      sync.Mutex
	entries []LogEntry
	size    int
	pos     int
	count   int
}

// NewRingBuffer creates a ring buffer with the given capacity.
func NewRingBuffer(size int) *RingBuffer {
	return &RingBuffer{
		entries: make([]LogEntry, size),
		size:    size,
	}
}

// Enabled implements slog.Handler — always enabled (filtering happens on read).
func (rb *RingBuffer) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

// Handle implements slog.Handler — stores the record in the ring buffer.
func (rb *RingBuffer) Handle(_ context.Context, r slog.Record) error {
	fields := make(map[string]string)
	r.Attrs(func(a slog.Attr) bool {
		fields[a.Key] = a.Value.String()
		return true
	})

	entry := LogEntry{
		Timestamp: r.Time,
		Level:     r.Level.String(),
		Message:   r.Message,
		Fields:    fields,
	}

	rb.mu.Lock()
	rb.entries[rb.pos] = entry
	rb.pos = (rb.pos + 1) % rb.size
	if rb.count < rb.size {
		rb.count++
	}
	rb.mu.Unlock()

	return nil
}

// WithAttrs implements slog.Handler — ring buffer doesn't carry attrs.
func (rb *RingBuffer) WithAttrs(_ []slog.Attr) slog.Handler {
	return rb
}

// WithGroup implements slog.Handler — ring buffer doesn't carry groups.
func (rb *RingBuffer) WithGroup(_ string) slog.Handler {
	return rb
}

// Entries returns log entries filtered by level, newest first.
// Pass empty level for all entries. limit <= 0 means no limit.
func (rb *RingBuffer) Entries(level string, limit int) []LogEntry {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.count == 0 {
		return []LogEntry{}
	}

	level = strings.ToUpper(level)
	var result []LogEntry

	// Walk backwards from most recent
	for i := 0; i < rb.count; i++ {
		idx := (rb.pos - 1 - i + rb.size) % rb.size
		entry := rb.entries[idx]

		if level != "" && entry.Level != level {
			continue
		}

		result = append(result, entry)

		if limit > 0 && len(result) >= limit {
			break
		}
	}

	if result == nil {
		return []LogEntry{}
	}
	return result
}
