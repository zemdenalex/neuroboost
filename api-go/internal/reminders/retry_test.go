package reminders

import (
	"testing"
	"time"
)

func TestShouldRetryWaitsForTheBackoff(t *testing.T) {
	now := time.Date(2026, 8, 11, 8, 0, 0, 0, time.UTC)
	failed := now.Add(-RetryBackoff + time.Second)
	if ShouldRetry(1, failed, now) {
		t.Error("retried before the backoff elapsed")
	}
	if !ShouldRetry(1, now.Add(-RetryBackoff), now) {
		t.Error("did not retry once the backoff had elapsed")
	}
}

func TestShouldRetryStopsAtTheAttemptCap(t *testing.T) {
	// 🔴 The whole reason attempts exists. Unbounded requeueing turns a user who
	// blocked the bot into one failed send a minute until somebody notices.
	now := time.Date(2026, 8, 11, 8, 0, 0, 0, time.UTC)
	long := now.Add(-24 * time.Hour)
	if !ShouldRetry(MaxDeliveryAttempts-1, long, now) {
		t.Error("gave up one attempt early")
	}
	if ShouldRetry(MaxDeliveryAttempts, long, now) {
		t.Error("retried past the cap")
	}
	if ShouldRetry(MaxDeliveryAttempts+7, long, now) {
		t.Error("retried well past the cap")
	}
}

func TestShouldRetryIgnoresAMissingOrFutureTimestamp(t *testing.T) {
	// Both would otherwise compute a backoff that has "already elapsed" and
	// retry on every single scan.
	now := time.Date(2026, 8, 11, 8, 0, 0, 0, time.UTC)
	if ShouldRetry(0, time.Time{}, now) {
		t.Error("retried a row with no failure timestamp")
	}
	if ShouldRetry(0, now.Add(time.Hour), now) {
		t.Error("retried a row that claims to have failed in the future")
	}
}

func TestFirstFailureIsEligibleOnceItIsDue(t *testing.T) {
	// The case that matters in practice: the morning digest fails at 08:00 and
	// must still arrive in the morning.
	now := time.Date(2026, 8, 11, 8, 6, 0, 0, time.UTC)
	failedAt := time.Date(2026, 8, 11, 8, 0, 0, 0, time.UTC)
	if !ShouldRetry(0, failedAt, now) {
		t.Error("a first failure was not retried")
	}
}
