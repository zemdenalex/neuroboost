package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	_ "time/tzdata" // embed the IANA tz database so recurrence DST handling works on any image

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"

	"neuroboost/api-go/internal/admin"
	"neuroboost/api-go/internal/config"
	"neuroboost/api-go/internal/database"
	"neuroboost/api-go/internal/logger"
	"neuroboost/api-go/internal/middleware"
	"neuroboost/api-go/internal/status"

	a "neuroboost/api-go/internal/auth"
	"neuroboost/api-go/internal/calendars"
	e "neuroboost/api-go/internal/events"
	exp "neuroboost/api-go/internal/export"
	f "neuroboost/api-go/internal/feedback"
	n "neuroboost/api-go/internal/needs"
	o "neuroboost/api-go/internal/opportunities"
	p "neuroboost/api-go/internal/patterns"
	pl "neuroboost/api-go/internal/planning"
	rfl "neuroboost/api-go/internal/reflections"
	rem "neuroboost/api-go/internal/reminders"
	t "neuroboost/api-go/internal/tasks"
	"neuroboost/api-go/internal/usersettings"
)

func main() {
	// Load config
	cfg := config.Load()

	// Initialize structured logger
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	logger.Init(logLevel)
	log := logger.Get()

	// Initialize database
	db, err := database.New(cfg.DatabaseURL)
	if err != nil {
		log.Error("Failed to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer db.Close()

	// Initialize package-level DB connections
	calendars.InitDB(db)
	e.InitDB(db)
	t.InitDB(db)
	rfl.InitDB(db)
	pl.InitDB(db)
	exp.InitDB(db)
	usersettings.InitDB(db)
	rem.InitDB(db)
	rem.InitService(log)

	// The reminder worker runs for the life of the process: it needs both the
	// pgx pool and the recurrence expander, so it cannot live in cron.
	workerCtx, stopWorker := context.WithCancel(context.Background())
	defer stopWorker()
	go rem.StartWorker(workerCtx, log)

	// Initialize router
	r := chi.NewRouter()

	// Middleware — request logging first, then CORS
	r.Use(middleware.RequestLogger(log))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{cfg.FrontendURL, "http://localhost:5173"},
		AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health check (no auth required)
	statusHandler := status.NewHandler(db)
	r.Get("/api/health", statusHandler.HealthHandler)

	// Auth and public feedback (no JWT required).
	//
	// Grouped rather than hung on the root router so the body limit can reach
	// them: these four decode a request body BEFORE any authentication, which
	// makes them the cheapest way to hand the process an unbounded allocation.
	// The API and the reminder worker share this process.
	authHandler := a.NewHandler(db, cfg)
	feedbackHandler := f.NewHandler(db)
	r.Group(func(pr chi.Router) {
		pr.Use(middleware.BodyLimit(middleware.PublicBodyLimit))

		pr.Post("/api/auth/telegram", authHandler.TelegramLogin)
		pr.Post("/api/auth/register", authHandler.Register)
		pr.Post("/api/auth/login", authHandler.Login)
		pr.Post("/api/auth/logout", authHandler.Logout)

		// Feedback - create is public (with optional auth)
		pr.Post("/api/feedback", feedbackHandler.Create)
	})

	// Service endpoints for the notifier bot. Guarded by a shared secret, NOT
	// by network topology — the notifier runs on a foreign host, so this is a
	// public endpoint returning every user's tg_id. Rate limit goes first so a
	// flood of bad-token requests is cheap to reject.
	r.Route("/api/svc", func(sr chi.Router) {
		sr.Use(rem.RateLimitMiddleware())
		sr.Use(middleware.BodyLimit(middleware.ServiceBodyLimit))
		sr.Use(rem.ServiceTokenMiddleware(cfg.ServiceToken))
		sr.Get("/notifications/pending", rem.PendingHandler)
		sr.Post("/notifications/{id}/ack", rem.AckHandler)
		// Button presses forwarded by the notifier. The user is identified by
		// tg_id inside the body, checked against the reminder's owner.
		sr.Post("/notifications/action", rem.ActionHandler)
	})

	// Protected routes (JWT required)
	r.Group(func(r chi.Router) {
		r.Use(middleware.JWTMiddleware(cfg.JWTSecret))
		r.Use(middleware.BodyLimit(middleware.AuthedBodyLimit))

		// Auth - get and update current user
		r.Get("/api/auth/me", authHandler.Me)
		r.Patch("/api/auth/me", authHandler.UpdateMe)

		// Feedback - list, update, and import require auth (admin check inside handlers)
		r.Get("/api/feedback", feedbackHandler.List)
		r.Patch("/api/feedback/{id}", feedbackHandler.Update)
		r.Post("/api/feedback/import", feedbackHandler.Import)

		// Admin endpoints (admin check inside handlers)
		adminHandler := admin.NewHandler(db)
		r.Get("/api/admin/health", adminHandler.Health)
		r.Get("/api/admin/logs", adminHandler.Logs)

		// Events
		r.Get("/api/events", e.ListHandler)
		r.Post("/api/events", e.CreateHandler)
		r.Get("/api/events/{id}", e.GetHandler)
		r.Patch("/api/events/{id}", e.UpdateHandler)
		r.Delete("/api/events/{id}", e.DeleteHandler)
		r.Patch("/api/events/{id}/move", e.MoveHandler)
		r.Patch("/api/events/{id}/resize", e.ResizeHandler)
		r.Post("/api/events/{id}/exceptions", e.AddExceptionHandler)
		r.Post("/api/events/{id}/reflection", rfl.CreateForEventHandler)

		// Tasks
		r.Get("/api/tasks", t.ListHandler)
		r.Post("/api/tasks", t.CreateHandler)
		r.Post("/api/tasks/batch", t.BatchCreateHandler)
		r.Get("/api/tasks/{id}", t.GetHandler)
		r.Patch("/api/tasks/{id}", t.UpdateHandler)
		r.Delete("/api/tasks/{id}", t.DeleteHandler)
		r.Post("/api/tasks/{id}/schedule", t.ScheduleHandler)
		r.Post("/api/tasks/{id}/log-time", t.LogTimeHandler)

		// Opportunities
		r.Get("/api/opportunities", o.ListHandler)
		r.Post("/api/opportunities", o.CreateHandler)
		r.Patch("/api/opportunities/{id}", o.UpdateHandler)
		r.Delete("/api/opportunities/{id}", o.DeleteHandler)

		// Needs
		r.Get("/api/needs", n.ListHandler)
		r.Post("/api/needs", n.CreateHandler)
		r.Patch("/api/needs/{id}", n.UpdateHandler)
		r.Delete("/api/needs/{id}", n.DeleteHandler)

		// Reflections
		r.Get("/api/reflections", rfl.ListHandler)
		r.Post("/api/reflections", rfl.CreateHandler)
		r.Patch("/api/reflections/{id}", rfl.UpdateHandler)

		// Patterns
		r.Get("/api/patterns/metrics", p.GetMetricsHandler)
		r.Get("/api/patterns/alert-status", p.GetAlertStatusHandler)

		// Planning
		r.Get("/api/planning/week", pl.GetWeekHandler)

		// Calendars
		r.Get("/api/calendars", calendars.ListHandler)
		r.Post("/api/calendars", calendars.CreateHandler)
		r.Patch("/api/calendars/{id}", calendars.UpdateHandler)
		r.Delete("/api/calendars/{id}", calendars.DeleteHandler)

		// Export (Import lives in its own group below — see there)
		r.Get("/api/export", exp.ExportHandler)
	})

	// POST /api/import, deliberately outside the group above.
	//
	// Its body is a whole user export — every event and task with descriptions
	// and tags — so the 1 MiB that fits the rest of the API would reject a
	// normal import. It cannot simply add a second, larger BodyLimit inside
	// that group either: chi runs the group's middleware first, so the 1 MiB
	// cap would already have wrapped the body and the wider limit would apply
	// to an already-truncated stream. Same JWT, its own ceiling.
	r.Group(func(ir chi.Router) {
		ir.Use(middleware.JWTMiddleware(cfg.JWTSecret))
		ir.Use(middleware.BodyLimit(middleware.ImportBodyLimit))
		ir.Post("/api/import", exp.ImportHandler)
	})

	// Start server
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8080"
	}

	log.Info("Starting NeuroBoost API",
		slog.String("port", port),
		slog.String("frontend_url", cfg.FrontendURL),
		slog.String("log_level", logLevel),
	)

	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Error("Server failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
