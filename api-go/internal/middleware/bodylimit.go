package middleware

import (
	"net/http"
)

// Body size limits, by exposure rather than by one global number.
//
// Before 2026-08-13 there was no limit anywhere: 25 json.NewDecoder call sites
// and zero http.MaxBytesReader. Four of them — register, login, telegram login,
// feedback — decode before any authentication, so a single curl with a
// multi-gigabyte body could pin the process. The API and the reminder worker
// share that process, which means the reminder delivery dies with it.
//
// One number cannot serve every route: /api/import legitimately carries a whole
// user export, and a limit tight enough to protect the login form would break
// the import silently — MaxBytesReader tears the stream and json.Decode reports
// that as invalid JSON, so the user sees 400 "Invalid request body" and has no
// way to know the body was simply too large.
const (
	// PublicBodyLimit covers the unauthenticated routes: an email, a password,
	// a feedback message. Generous for all three.
	PublicBodyLimit int64 = 64 << 10 // 64 KiB

	// ServiceBodyLimit covers /api/svc/*, whose bodies the bot writes: an id, a
	// tg_id, an action, a minute count.
	ServiceBodyLimit int64 = 32 << 10 // 32 KiB

	// AuthedBodyLimit covers the JWT-protected group — events, tasks,
	// reflections, calendars, settings. Far larger than any single record.
	AuthedBodyLimit int64 = 1 << 20 // 1 MiB

	// ImportBodyLimit is the deliberate exception: POST /api/import carries an
	// entire export, events and tasks with descriptions and tags. Roughly 10k
	// events at ~1 KB each, with room to spare.
	ImportBodyLimit int64 = 32 << 20 // 32 MiB
)

// BodyLimit caps how much of a request body a handler can read.
//
// The cap is enforced by http.MaxBytesReader, which also tells the server to
// close the connection rather than drain a body nobody will read. Handlers that
// want to answer 413 instead of 400 should check for *http.MaxBytesError with
// errors.As — see util.RespondDecodeError.
func BodyLimit(n int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// GET and DELETE carry no body worth capping, and wrapping them
			// costs an allocation per request for nothing.
			if r.Body != nil && r.Body != http.NoBody {
				r.Body = http.MaxBytesReader(w, r.Body, n)
			}
			next.ServeHTTP(w, r)
		})
	}
}
