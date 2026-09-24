package logger

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

// The buffer keeps everything (filtered on read), but stdout must keep its own
// level: a DEBUG record used to reach the info-level text handler because Handle
// fanned out to every handler without asking it (audit 24.09, A7).
func TestMultiHandlerHonoursEachHandlersLevel(t *testing.T) {
	var out bytes.Buffer
	text := slog.NewTextHandler(&out, &slog.HandlerOptions{Level: slog.LevelInfo})
	ring := NewRingBuffer(10)
	log := slog.New(&multiHandler{handlers: []slog.Handler{text, ring}})

	log.Debug("quiet detail")
	log.Info("visible line")

	if strings.Contains(out.String(), "quiet detail") {
		t.Fatalf("a DEBUG record reached the info-level output:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "visible line") {
		t.Fatalf("the INFO record is missing from the output:\n%s", out.String())
	}
	if n := len(ring.Entries("", 10)); n != 2 {
		t.Fatalf("buffer holds %d entries, want both (it filters on read)", n)
	}
}
