package reflections

import (
    "net/http"
    u "neuroboost/api-go/internal/util"
)

func ListHandler(w http.ResponseWriter, r *http.Request) {
    u.Write501(w, "List Reflections", "GET /api/reflections")
}

func CreateHandler(w http.ResponseWriter, r *http.Request) {
    u.Write501(w, "Create Reflection", "POST /api/reflections")
}

func UpdateHandler(w http.ResponseWriter, r *http.Request) {
    u.Write501(w, "Update Reflection", "PATCH /api/reflections/:id")
}
