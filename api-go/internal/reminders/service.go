package reminders

import (
	"crypto/subtle"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"neuroboost/api-go/internal/util"
)

var svcLog *slog.Logger

// InitService gives the service endpoints their logger. Rejected requests are
// logged with the caller IP because this prefix is reachable from the public
// internet once the notifier moves to a foreign host.
func InitService(log *slog.Logger) { svcLog = log }

// ServiceTokenMiddleware guards the /api/svc prefix with a shared secret.
//
// This is NOT an internal endpoint, whatever it is called. The bot currently
// reaches the API over the compose network, but after the move abroad this is
// a public HTTPS endpoint handing out every user's tg_id and message text,
// and the token is the only thing in the way. Hence the /api/svc prefix rather
// than /api/internal: nobody should read the name and assume network isolation.
func ServiceTokenMiddleware(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Unset means closed, never open. A deployment that forgets
			// SERVICE_TOKEN must fail loudly rather than serve everything to
			// anyone who sends an empty header.
			if token == "" {
				util.RespondError(w, http.StatusServiceUnavailable, "SERVICE_DISABLED",
					"Service endpoints are not configured")
				return
			}
			presented := r.Header.Get("X-Service-Token")
			// Constant time: a byte-by-byte == leaks the token through response
			// timing to anyone who can reach this endpoint.
			if subtle.ConstantTimeCompare([]byte(presented), []byte(token)) != 1 {
				if svcLog != nil {
					svcLog.Warn("service token rejected",
						slog.String("ip", r.RemoteAddr),
						slog.String("path", r.URL.Path))
				}
				util.RespondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid service token")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// rateLimiter is a fixed-window counter over the whole /api/svc prefix.
type rateLimiter struct {
	mu       sync.Mutex
	count    int
	window   time.Time
	limit    int
	duration time.Duration
}

func newRateLimiter(limit int, d time.Duration) *rateLimiter {
	return &rateLimiter{limit: limit, duration: d}
}

func (rl *rateLimiter) allow(now time.Time) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if now.Sub(rl.window) >= rl.duration {
		rl.window = now
		rl.count = 0
	}
	rl.count++
	return rl.count <= rl.limit
}

// svcRateLimit caps the prefix at 60 requests a minute. The legitimate caller
// is one notifier polling once a minute; anything near this is a bug or an
// attack.
const svcRateLimit = 60

// RateLimitMiddleware throttles the /api/svc prefix. Register it BEFORE the
// token check so a flood of bad-token requests is cheap to reject.
func RateLimitMiddleware() func(http.Handler) http.Handler {
	rl := newRateLimiter(svcRateLimit, time.Minute)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !rl.allow(time.Now()) {
				if svcLog != nil {
					svcLog.Warn("service rate limit hit", slog.String("ip", r.RemoteAddr))
				}
				util.RespondError(w, http.StatusTooManyRequests, "RATE_LIMITED", "Too many requests")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// PendingNotification is what the notifier receives.
type PendingNotification struct {
	ID         string `json:"id"`
	TgID       int64  `json:"tg_id"`
	Text       string `json:"text"`
	SourceKind string `json:"source_kind"`
	SourceID   string `json:"source_id"`
}

const (
	pendingBatchLimit = 100
	// A row handed to the notifier but never acked goes back to PENDING after
	// this long — the notifier died between claim and ack.
	sendingStaleAfter = 5 * time.Minute
)

// PendingHandler claims up to 100 due notifications and marks them SENDING.
func PendingHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Reclaim rows a dead notifier left behind before claiming new ones. The
	// interval is built from an integer minute count rather than a Go duration
	// string: "5m0s" happens to parse as a Postgres interval, but that is a
	// coincidence of two formats, not a contract.
	if _, err := db.Pool.Exec(ctx, `
		UPDATE reminder SET status = 'PENDING'
		WHERE status = 'SENDING' AND sent_at < NOW() - make_interval(mins => $1)`,
		int(sendingStaleAfter.Minutes())); err != nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to reclaim stale reminders")
		return
	}

	// Claim and return in one statement so two notifier instances cannot claim
	// the same row.
	//
	// Two comma-separated FROM items, NOT a JOIN: in an UPDATE ... FROM, the
	// update target `r` is not in scope for a JOIN's ON clause.
	//
	// u.tg_id IS NOT NULL is repeated here even though the scan already filters
	// on it — the scan filters at write time, and a user can clear their tg_id
	// between scan and claim. tg_id is nullable and TgID is a plain int64, so
	// without this the row scan fails.
	//
	// remind_at <= NOW() is essential: a snooze row is created with remind_at
	// in the future, and without this filter it would go out on creation.
	rows, err := db.Pool.Query(ctx, `
		UPDATE reminder r
		SET status = 'SENDING', sent_at = NOW()
		FROM (
			SELECT id FROM reminder
			WHERE status = 'PENDING' AND remind_at <= NOW()
			ORDER BY remind_at
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		) claimed,
		"user" u
		WHERE r.id = claimed.id
		  AND u.id = r.user_id
		  AND u.tg_id IS NOT NULL
		RETURNING r.id, u.tg_id, COALESCE(r.message, ''), r.source_kind,
		          COALESCE(r.event_id::text, r.task_id::text, '')`,
		pendingBatchLimit)
	if err != nil {
		if svcLog != nil {
			svcLog.Error("failed to claim reminders", slog.String("error", err.Error()))
		}
		util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to claim reminders")
		return
	}
	defer rows.Close()

	out := []PendingNotification{}
	for rows.Next() {
		var n PendingNotification
		if err := rows.Scan(&n.ID, &n.TgID, &n.Text, &n.SourceKind, &n.SourceID); err != nil {
			util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to read reminders")
			return
		}
		out = append(out, n)
	}
	if err := rows.Err(); err != nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to read reminders")
		return
	}

	util.RespondJSON(w, http.StatusOK, out)
}

type ackRequest struct {
	Delivered bool   `json:"delivered"`
	Error     string `json:"error,omitempty"`
}

// AckHandler records the outcome of one delivery attempt.
func AckHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		util.RespondError(w, http.StatusBadRequest, "MISSING_ID", "Reminder ID is required")
		return
	}

	var req ackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.RespondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}

	status := "FAILED"
	if req.Delivered {
		status = "SENT"
	}
	if _, err := db.Pool.Exec(r.Context(),
		`UPDATE reminder SET status = $1, sent_at = NOW() WHERE id = $2`,
		status, id); err != nil {
		util.RespondError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to record delivery")
		return
	}
	if !req.Delivered && svcLog != nil {
		svcLog.Warn("notification delivery failed",
			slog.String("reminder_id", id), slog.String("error", req.Error))
	}
	util.RespondJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
