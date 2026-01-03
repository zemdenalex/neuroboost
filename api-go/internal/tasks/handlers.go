package tasks

import (
	"net/http"

	"neuroboost/api-go/internal/util"
)

func ListHandler(w http.ResponseWriter, r *http.Request) {
	util.Write501(w, "List Tasks", "GET /api/tasks")
}

func CreateHandler(w http.ResponseWriter, r *http.Request) {
	util.Write501(w, "Create Task", "POST /api/tasks")
}

func GetHandler(w http.ResponseWriter, r *http.Request) {
	util.Write501(w, "Get Task", "GET /api/tasks/:id")
}

func UpdateHandler(w http.ResponseWriter, r *http.Request) {
	util.Write501(w, "Update Task", "PATCH /api/tasks/:id")
}

func DeleteHandler(w http.ResponseWriter, r *http.Request) {
	util.Write501(w, "Delete Task", "DELETE /api/tasks/:id")
}

func ScheduleHandler(w http.ResponseWriter, r *http.Request) {
	util.Write501(w, "Schedule Task", "POST /api/tasks/:id/schedule")
}
