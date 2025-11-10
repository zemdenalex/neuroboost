package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	_ "github.com/lib/pq"

	"github.com/zemdenalex/neuroboost/internal/auth"
	"github.com/zemdenalex/neuroboost/internal/config"
	mw "github.com/zemdenalex/neuroboost/internal/middleware"
)

func main() {
	config.LoadDotEnvOnce()

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

	auth.Init(db)

	r := chi.NewRouter()
	r.Use(chimw.RequestID, chimw.RealIP, chimw.Recoverer, chimw.Timeout(60*time.Second))

	// Optional CORS (needed if your frontend runs on another origin and uses cookies)
	if os.Getenv("ENABLE_CORS") == "1" {
		frontend := os.Getenv("FRONTEND_ORIGIN") // e.g., http://localhost:5173
		if frontend == "" {
			frontend = "http://localhost:5173"
		}
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins:   []string{frontend},
			AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
			ExposedHeaders:   []string{"Link"},
			AllowCredentials: true,
			MaxAge:           300,
		}))
	}

	// Health
	r.Get("/api/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	// Public
	r.Post("/api/auth/login", auth.LoginHandler)

	// Protected
	r.Group(func(pr chi.Router) {
		pr.Use(mw.JWTMiddleware(func(id int64) (*auth.User, error) {
			return auth.GetUserByID(db, id)
		}))

		pr.Get("/api/auth/me", auth.MeHandler)
		pr.Post("/api/auth/logout", auth.LogoutHandler)
		pr.Post("/api/auth/refresh", auth.RefreshHandler)

		// ... other protected routes ...
	})

	addr := ":8080"
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatal(err)
	}
}
