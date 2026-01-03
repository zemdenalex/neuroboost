package planning

import (
	"net/http"

	"neuroboost/api-go/internal/util"
)

func GetGraphHandler(w http.ResponseWriter, r *http.Request) {
	util.Write501(w, "Get Planning Graph", "GET /api/planning/graph")
}

func CreateNodeHandler(w http.ResponseWriter, r *http.Request) {
	util.Write501(w, "Create Planning Node", "POST /api/planning/nodes")
}

func CreateEdgeHandler(w http.ResponseWriter, r *http.Request) {
	util.Write501(w, "Create Planning Edge", "POST /api/planning/edges")
}
