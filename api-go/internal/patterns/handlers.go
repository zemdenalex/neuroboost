package patterns

import (
	"net/http"

	"neuroboost/api-go/internal/util"
)

func GetMetricsHandler(w http.ResponseWriter, r *http.Request) {
	util.Write501(w, "Get Pattern Metrics", "GET /api/patterns/metrics")
}

func GetAlertStatusHandler(w http.ResponseWriter, r *http.Request) {
	util.Write501(w, "Get Alert Status", "GET /api/patterns/alert-status")
}
