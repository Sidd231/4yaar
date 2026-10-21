package utils

import (
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

// NewWebSocketUpgrader creates a websocket.Upgrader that validates incoming Origin headers
// against a configurable allowlist.
// If allowedOrigins contains "*", all origins are permitted.
// If Origin header is empty (e.g. non-browser clients, native apps, backend tests), the connection is allowed.
func NewWebSocketUpgrader(allowedOrigins []string) websocket.Upgrader {
	return websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" {
				// Non-browser clients, automated scripts, or test harnesses typically omit Origin
				return true
			}
			for _, allowed := range allowedOrigins {
				if allowed == "*" || strings.EqualFold(origin, allowed) {
					return true
				}
			}
			return false
		},
	}
}

// ExtractWebSocketToken retrieves the JWT bearer token for WebSocket authentication.
// It prioritizes reading from the standard 'Authorization' header ('Bearer <token>') for
// clients that support custom headers on the HTTP upgrade request.
//
// API/CLIENT CONSTRAINT NOTE:
// Standard browser WebSocket APIs ('new WebSocket(url)') cannot set custom HTTP headers
// during the handshake. To maintain compatibility with existing browser clients without breaking
// the contract, this function falls back to reading the 'token' query parameter.
//
// SECURITY CONSTRAINT:
// Bearer tokens received via query parameters MUST NEVER be logged, written to stdout, or
// included in application error strings or access logs.
func ExtractWebSocketToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return strings.TrimSpace(parts[1])
		}
	}

	return r.URL.Query().Get("token")
}
