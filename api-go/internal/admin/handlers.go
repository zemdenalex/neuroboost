package admin

import (
	"context"
	"net/http"
	"runtime"
	"strconv"
	"time"

	"neuroboost/api-go/internal/database"
	"neuroboost/api-go/internal/logger"
	"neuroboost/api-go/internal/middleware"
	"neuroboost/api-go/internal/util"
)

const version = "0.4.0"

var startTime = time.Now()

type Handler struct {
	db *database.DB
}

func NewHandler(db *database.DB) *Handler {
	return &Handler{db: db}
}

// HealthResponse contains system health information.
type HealthResponse struct {
	Version          string `json:"version"`
	UptimeSeconds    int64  `json:"uptime_seconds"`
	GoVersion        string `json:"go_version"`
	DBStatus         string `json:"db_status"`
	DBPingMs         int64  `json:"db_ping_ms"`
	MigrationVersion int    `json:"migration_version"`
	MemoryMB         uint64 `json:"memory_mb"`
	Goroutines       int    `json:"goroutines"`
}

// Health handles GET /api/admin/health
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		util.RespondError(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Authentication required")
		return
	}

	isAdmin, err := h.isUserAdmin(r.Context(), userID)
	if err != nil || !isAdmin {
		util.RespondError(w, http.StatusForbidden, "NOT_ADMIN", "Admin access required")
		return
	}

	// DB ping
	pingStart := time.Now()
	dbStatus := "ok"
	if err := h.db.Ping(r.Context()); err != nil {
		dbStatus = "error: " + err.Error()
	}
	pingMs := time.Since(pingStart).Milliseconds()

	// Migration version
	migrationVersion := 0
	_ = h.db.Pool.QueryRow(r.Context(),
		`SELECT version FROM schema_migrations LIMIT 1`,
	).Scan(&migrationVersion)

	// Memory stats
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	resp := HealthResponse{
		Version:          version,
		UptimeSeconds:    int64(time.Since(startTime).Seconds()),
		GoVersion:        runtime.Version(),
		DBStatus:         dbStatus,
		DBPingMs:         pingMs,
		MigrationVersion: migrationVersion,
		MemoryMB:         mem.Alloc / 1024 / 1024,
		Goroutines:       runtime.NumGoroutine(),
	}

	util.RespondJSON(w, http.StatusOK, resp)
}

// Logs handles GET /api/admin/logs
func (h *Handler) Logs(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		util.RespondError(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Authentication required")
		return
	}

	isAdmin, err := h.isUserAdmin(r.Context(), userID)
	if err != nil || !isAdmin {
		util.RespondError(w, http.StatusForbidden, "NOT_ADMIN", "Admin access required")
		return
	}

	buf := logger.Buffer()
	if buf == nil {
		util.RespondJSON(w, http.StatusOK, []logger.LogEntry{})
		return
	}

	level := r.URL.Query().Get("level")
	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	entries := buf.Entries(level, limit)
	util.RespondJSON(w, http.StatusOK, entries)
}

func (h *Handler) isUserAdmin(ctx context.Context, userID string) (bool, error) {
	var isAdmin bool
	err := h.db.Pool.QueryRow(ctx, `SELECT COALESCE(is_admin, FALSE) FROM "user" WHERE id = $1`, userID).Scan(&isAdmin)
	return isAdmin, err
}
