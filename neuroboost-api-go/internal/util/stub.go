package util

import (
    "encoding/json"
    "net/http"
)

type Stub struct {
    Error    string `json:"error"`
    Feature  string `json:"feature"`
    Endpoint string `json:"endpoint"`
}

func Write501(w http.ResponseWriter, feature, endpoint string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusNotImplemented)
    _ = json.NewEncoder(w).Encode(Stub{
        Error:    "Not implemented yet",
        Feature:  feature,
        Endpoint: endpoint,
    })
}
