package util

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// Response is the standard API response wrapper
type Response struct {
	Data  interface{} `json:"data,omitempty"`
	Error *APIError   `json:"error,omitempty"`
	Meta  *Meta       `json:"meta,omitempty"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Meta struct {
	Total int `json:"total,omitempty"`
	Page  int `json:"page,omitempty"`
	Limit int `json:"limit,omitempty"`
}

// RespondJSON sends a JSON response with data
func RespondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{Data: data})
}

// RespondError sends a JSON error response
func RespondError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{
		Error: &APIError{
			Code:    code,
			Message: message,
		},
	})
}

// RespondDecodeError answers a failed json.Decode of a request body.
//
// It exists to separate two failures that json.Decode reports identically. When
// middleware.BodyLimit caps a body, http.MaxBytesReader tears the stream, and
// the decoder sees truncated input — indistinguishable, at the call site, from
// a genuinely malformed payload. Answering both with 400 "Invalid request body"
// is how an import that is merely too large gets reported as broken JSON, which
// tells the user nothing about what to do next.
//
// 413 says the body was too large and names the ceiling; everything else keeps
// the previous 400.
func RespondDecodeError(w http.ResponseWriter, err error) {
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		RespondError(w, http.StatusRequestEntityTooLarge, "BODY_TOO_LARGE",
			fmt.Sprintf("Request body exceeds the %d byte limit for this endpoint", maxErr.Limit))
		return
	}
	RespondError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
}

// Write501 sends a 501 Not Implemented response (for stubs)
func Write501(w http.ResponseWriter, feature, endpoint string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{
		"error":    "Not implemented yet",
		"feature":  feature,
		"endpoint": endpoint,
	})
}
