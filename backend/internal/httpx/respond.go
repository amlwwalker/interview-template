// Package httpx holds the small, boring helpers that keep handlers readable:
// one way to write JSON, one shape for errors.
package httpx

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// ErrorBody is the single error shape the API returns. Having exactly one
// means the frontend never has to guess.
type ErrorBody struct {
	Error   string            `json:"error"`
	Details map[string]string `json:"details,omitempty"`
}

// JSON writes v as a JSON response with the given status code.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if v == nil {
		return
	}

	if err := json.NewEncoder(w).Encode(v); err != nil {
		// The status line is already sent, so all we can do is record it.
		slog.Error("failed to encode response", "error", err)
	}
}

// Error writes a structured error response.
func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, ErrorBody{Error: message})
}

// ValidationError writes a 422 with per-field messages.
func ValidationError(w http.ResponseWriter, details map[string]string) {
	JSON(w, http.StatusUnprocessableEntity, ErrorBody{
		Error:   "validation failed",
		Details: details,
	})
}

// NoContent writes a bare 204.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}
