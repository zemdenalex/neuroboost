package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/lib/pq"

	"github.com/zemdenalex/neuroboost/internal/auth"
	mw "github.com/zemdenalex/neuroboost/internal/middleware"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("db ping: %v", err)
	}

	// Initialize auth package globals (db + bot token + env)
	auth.Init(db)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Health
	r.Get("/api/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	// Public routes
	r.Post("/api/auth/login", auth.LoginHandler)

	// Protected routes
	r.Group(func(pr chi.Router) {
		pr.Use(mw.JWTMiddleware(func(id int64) (*auth.User, error) {
			return auth.GetUserByID(db, id)
		}))

		pr.Get("/api/auth/me", auth.MeHandler)
		pr.Post("/api/auth/logout", auth.LogoutHandler)
		pr.Post("/api/auth/refresh", auth.RefreshHandler)

		// ... other protected routes here ...
	})

	addr := ":8080"
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatal(err)
	}
}
