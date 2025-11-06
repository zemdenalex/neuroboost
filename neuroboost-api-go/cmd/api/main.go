package main

import (
    "net/http"
    "github.com/go-chi/chi/v5"
    "github.com/go-chi/cors"
    "neuroboost/api-go/internal/middleware"
    "neuroboost/api-go/internal/status"
    a "neuroboost/api-go/internal/auth"
    e "neuroboost/api-go/internal/events"
    t "neuroboost/api-go/internal/tasks"
    o "neuroboost/api-go/internal/opportunities"
    n "neuroboost/api-go/internal/needs"
    rfl "neuroboost/api-go/internal/reflections"
    p "neuroboost/api-go/internal/patterns"
    pl "neuroboost/api-go/internal/planning"
)

func main() {
    r := chi.NewRouter()
    r.Use(cors.Handler(cors.Options{
        AllowedOrigins:   []string{"*"},
        AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
        AllowedHeaders:   []string{"*"},
        AllowCredentials: true,
        MaxAge:           300,
    }))

    // Health (works)
    r.Get("/api/health", status.HealthHandler)

    // Auth (no JWT)
    r.Post("/api/auth/telegram", a.LoginHandler)
    r.Get("/api/auth/me", a.MeHandler)
    r.Post("/api/auth/logout", a.LogoutHandler)

    // Protected (JWT stub)
    r.Group(func(r chi.Router) {
        r.Use(middleware.JWTMiddleware)

        // Events
        r.Get("/api/events", e.ListHandler)
        r.Post("/api/events", e.CreateHandler)
        r.Get("/api/events/{id}", e.GetHandler)
        r.Patch("/api/events/{id}", e.UpdateHandler)
        r.Delete("/api/events/{id}", e.DeleteHandler)
        r.Patch("/api/events/{id}/move", e.MoveHandler)
        r.Patch("/api/events/{id}/resize", e.ResizeHandler)
        r.Post("/api/events/{id}/exceptions", e.AddExceptionHandler)

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

    http.ListenAndServe(":8080", r)
}
