package utils

import (
	"encoding/json"
	"net/http"
)

// APIResponse represents the standardized JSON envelope.
type APIResponse struct {
	Data  any     `json:"data"`
	Error *string `json:"error"`
}

// JSONResponse writes a successful or error JSON response matching { "data": ..., "error": ... }.
func JSONResponse(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(APIResponse{
		Data:  data,
		Error: nil,
	})
}

// JSONError writes an error response matching { "data": null, "error": "message" }.
func JSONError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(APIResponse{
		Data:  nil,
		Error: &message,
	})
}
