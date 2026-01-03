package auth

import (
    "net/http"
    u "neuroboost/api-go/internal/util"
)

func LoginHandler(w http.ResponseWriter, r *http.Request) {
    u.Write501(w, "Telegram Authentication", "POST /api/auth/telegram")
}

func MeHandler(w http.ResponseWriter, r *http.Request) {
    u.Write501(w, "Get Current User", "GET /api/auth/me")
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
    u.Write501(w, "Logout", "POST /api/auth/logout")
}
