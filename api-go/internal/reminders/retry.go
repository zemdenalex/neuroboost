package reminders

import "time"

// Retry policy for a delivery that failed.
//
// Before this existed a FAILED row was terminal. idx_reminder_dedupe stops the
// scan from re-creating an identical row, so one failure lost that reminder for
// the whole local day — which is exactly how the empty-digest bug stayed
// invisible: the send failed every morning and nothing retried it.
const (
	// MaxDeliveryAttempts bounds the retries. Unbounded requeueing turns a user
	// who blocked the bot into one failed send per minute, forever.
	MaxDeliveryAttempts = 3
	// RetryBackoff is how long a failed row waits before going back in the
	// queue. Long enough that a Telegram outage is over, short enough that a
	// morning digest still arrives in the morning.
	RetryBackoff = 5 * time.Minute
)

// ShouldRetry reports whether a failed delivery goes back into the queue.
//
// Pure so the policy can be tested without a database — the SQL that applies it
// is a single UPDATE, and the interesting part is the decision, not the query.
func ShouldRetry(attempts int, failedAt time.Time, now time.Time) bool {
	if attempts >= MaxDeliveryAttempts {
		return false
	}
	// A row whose timestamp is missing or in the future would otherwise retry
	// immediately and repeatedly; treat it as not yet due.
	if failedAt.IsZero() || failedAt.After(now) {
		return false
	}
	return !now.Before(failedAt.Add(RetryBackoff))
}
