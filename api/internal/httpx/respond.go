// Package httpx holds small shared HTTP response helpers used by handlers.
package httpx

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// WriteJSON encodes v as the JSON response body with the given status code.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("failed to encode json response", "error", err)
	}
}

// ErrorResponse is the standard error body shape used across the API, e.g.
// {"error": "validation_error", "message": "..."}.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// WriteError writes a standard ErrorResponse body with the given status.
func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, ErrorResponse{Error: code, Message: message})
}
