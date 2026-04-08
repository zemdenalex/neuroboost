package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"

	"neuroboost/api-go/internal/admin"
	"neuroboost/api-go/internal/config"
	"neuroboost/api-go/internal/database"
	"neuroboost/api-go/internal/logger"
	"neuroboost/api-go/internal/middleware"
	"neuroboost/api-go/internal/status"

	a "neuroboost/api-go/internal/auth"
	e "neuroboost/api-go/internal/events"
	f "neuroboost/api-go/internal/feedback"
	n "neuroboost/api-go/internal/needs"
	o "neuroboost/api-go/internal/opportunities"
	p "neuroboost/api-go/internal/patterns"
	pl "neuroboost/api-go/internal/planning"
	rfl "neuroboost/api-go/internal/reflections"
	t "neuroboost/api-go/internal/tasks"
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
	e.InitDB(db)
	t.InitDB(db)
	rfl.InitDB(db)

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

	// Auth routes (no JWT required)
	authHandler := a.NewHandler(db, cfg)
	r.Post("/api/auth/telegram", authHandler.TelegramLogin)
	r.Post("/api/auth/register", authHandler.Register)
	r.Post("/api/auth/login", authHandler.Login)
	r.Post("/api/auth/logout", authHandler.Logout)

	// Feedback - create is public (with optional auth)
	feedbackHandler := f.NewHandler(db)
	r.Post("/api/feedback", feedbackHandler.Create)

	// Protected routes (JWT required)
	r.Group(func(r chi.Router) {
		r.Use(middleware.JWTMiddleware(cfg.JWTSecret))

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
		r.Get("/api/tasks/{id}", t.GetHandler)
		r.Patch("/api/tasks/{id}", t.UpdateHandler)
		r.Delete("/api/tasks/{id}", t.DeleteHandler)
		r.Post("/api/tasks/{id}/schedule", t.ScheduleHandler)

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

		// Planning graph
		r.Get("/api/planning/graph", pl.GetGraphHandler)
		r.Post("/api/planning/nodes", pl.CreateNodeHandler)
		r.Post("/api/planning/edges", pl.CreateEdgeHandler)
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
