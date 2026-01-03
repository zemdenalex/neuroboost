package status

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"neuroboost/api-go/internal/database"
)

type Handler struct {
	db *database.DB
}

func NewHandler(db *database.DB) *Handler {
	return &Handler{db: db}
}

type HealthResponse struct {
	Status   string `json:"status"`
	Service  string `json:"service"`
	Version  string `json:"version"`
	Database string `json:"database"`
}

func (h *Handler) HealthHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	dbStatus := "connected"
	if err := h.db.Ping(ctx); err != nil {
		dbStatus = "disconnected"
	}

	status := "ok"
	if dbStatus != "connected" {
		status = "degraded"
	}

	resp := HealthResponse{
		Status:   status,
		Service:  "neuroboost-api",
		Version:  "0.4.0",
		Database: dbStatus,
	}

	w.Header().Set("Content-Type", "application/json")
	if status != "ok" {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	json.NewEncoder(w).Encode(resp)
}
