package tasks

import (
	"os"
	"testing"
	"time"
)

// TestMain runs this package's tests on the clock CI has: UTC.
//
// 🔴 Seven tests here were green on the developer's machine and red on CI
// every night from 21:00 UTC (22.09). They built «tomorrow» from time.Now(),
// whose zone is the MACHINE's. The machine is on Moscow time, like the test
// user, so its tomorrow always agreed; the runner is UTC, and for three hours
// a night its tomorrow is the user's today. A test that passes only in the
// author's zone has not been run, and TZ=UTC does not help on Windows — Go
// ignores it there. Pinning time.Local makes every run the runner's run.
func TestMain(m *testing.M) {
	time.Local = time.UTC
	os.Exit(m.Run())
}

// userToday is today's date where the seeded test user lives.
//
// seedTaskUser writes no timezone, so the API reads the default,
// Europe/Moscow. Every «today» and «tomorrow» in these tests is that user's,
// never the clock's — the question learning-the-right-time-in-the-wrong-zone
// asks of the code, asked of the tests.
func userToday() time.Time {
	return LocalDay(time.Now(), "Europe/Moscow")
}
