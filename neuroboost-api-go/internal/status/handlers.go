package status

import (
    "encoding/json"
    "net/http"
)

func HealthHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    _ = json.NewEncoder(w).Encode(map[string]any{
        "status":  "ok",
        "service": "neuroboost-api",
        "version": "0.4.0-skeleton",
    })
}
