package events

import (
    "net/http"
    u "neuroboost/api-go/internal/util"
)

func ListHandler(w http.ResponseWriter, r *http.Request) {
    u.Write501(w, "List Events", "GET /api/events")
}

func CreateHandler(w http.ResponseWriter, r *http.Request) {
    u.Write501(w, "Create Event", "POST /api/events")
}

func GetHandler(w http.ResponseWriter, r *http.Request) {
    u.Write501(w, "Get Event", "GET /api/events/:id")
}

func UpdateHandler(w http.ResponseWriter, r *http.Request) {
    u.Write501(w, "Update Event", "PATCH /api/events/:id")
}

func DeleteHandler(w http.ResponseWriter, r *http.Request) {
    u.Write501(w, "Delete Event", "DELETE /api/events/:id")
}

func MoveHandler(w http.ResponseWriter, r *http.Request) {
    u.Write501(w, "Move Event (DnD)", "PATCH /api/events/:id/move")
}

func ResizeHandler(w http.ResponseWriter, r *http.Request) {
    u.Write501(w, "Resize Event (DnD)", "PATCH /api/events/:id/resize")
}

func AddExceptionHandler(w http.ResponseWriter, r *http.Request) {
    u.Write501(w, "Add Event Exception", "POST /api/events/:id/exceptions")
}
