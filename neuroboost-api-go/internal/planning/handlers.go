package planning

import (
    "net/http"
    u "neuroboost/api-go/internal/util"
)

func GetGraphHandler(w http.ResponseWriter, r *http.Request) {
    u.Write501(w, "Get Planning Graph", "GET /api/planning/graph")
}

func CreateNodeHandler(w http.ResponseWriter, r *http.Request) {
    u.Write501(w, "Create Planning Node", "POST /api/planning/nodes")
}

func CreateEdgeHandler(w http.ResponseWriter, r *http.Request) {
    u.Write501(w, "Create Planning Edge", "POST /api/planning/edges")
}
