package reminders

import (
	"context"
	"log/slog"
	"runtime/debug"
	"time"
)

const (
	tickInterval = time.Minute
	// The scan window deliberately overlaps previous ticks: a missed tick
	// (deploy, restart, GC pause) must not lose a reminder. The unique index
	// makes re-scanning the same minute free.
	windowBehind = 2 * time.Minute
	windowAhead  = 1 * time.Minute
)

// runScan performs one tick and never panics out of it.
//
// The recover is load-bearing, not defensive decoration: this worker shares a
// process with the API, and an unrecovered panic on a goroutine terminates the
// whole program — so one malformed event would take the entire backend down,
// every sixty seconds, until someone noticed. A dropped scan is recoverable
// (the next window overlaps this one); a dead API is not.
func runScan(ctx context.Context, now time.Time, log *slog.Logger) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("reminder scan panicked",
				slog.Any("panic", r),
				slog.String("stack", string(debug.Stack())))
		}
	}()

	n, err := Scan(ctx, now.Add(-windowBehind), now.Add(windowAhead), log)
	if err != nil {
		log.Error("reminder scan failed", slog.String("error", err.Error()))
		return
	}
	if n > 0 {
		log.Info("reminders scheduled", slog.Int("count", n))
	}
}

// StartWorker runs the reminder scan once a minute until ctx is cancelled.
// It lives inside the API process rather than in cron because it needs both
// the pgx pool and events.OccurrencesInRange, and both live here.
func StartWorker(ctx context.Context, log *slog.Logger) {
	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()

	log.Info("reminder worker started", slog.String("interval", tickInterval.String()))
	for {
		select {
		case <-ctx.Done():
			log.Info("reminder worker stopped")
			return
		case now := <-ticker.C:
			runScan(ctx, now, log)
		}
	}
}
