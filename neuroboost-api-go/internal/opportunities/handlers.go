package opportunities

import (
    "net/http"
    u "neuroboost/api-go/internal/util"
)

func ListHandler(w http.ResponseWriter, r *http.Request) {
    u.Write501(w, "List Opportunities", "GET /api/opportunities")
}

func CreateHandler(w http.ResponseWriter, r *http.Request) {
    u.Write501(w, "Create Opportunity", "POST /api/opportunities")
}

func UpdateHandler(w http.ResponseWriter, r *http.Request) {
    u.Write501(w, "Update Opportunity", "PATCH /api/opportunities/:id")
}

func DeleteHandler(w http.ResponseWriter, r *http.Request) {
    u.Write501(w, "Delete Opportunity", "DELETE /api/opportunities/:id")
}
