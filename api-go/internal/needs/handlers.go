package needs

import (
	"net/http"

	"neuroboost/api-go/internal/util"
)

func ListHandler(w http.ResponseWriter, r *http.Request) {
	util.Write501(w, "List Needs", "GET /api/needs")
}

func CreateHandler(w http.ResponseWriter, r *http.Request) {
	util.Write501(w, "Create Need", "POST /api/needs")
}

func UpdateHandler(w http.ResponseWriter, r *http.Request) {
	util.Write501(w, "Update Need", "PATCH /api/needs/:id")
}

func DeleteHandler(w http.ResponseWriter, r *http.Request) {
	util.Write501(w, "Delete Need", "DELETE /api/needs/:id")
}
