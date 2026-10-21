package utils

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWebSocketUpgrader_OriginCheck(t *testing.T) {
	allowedOrigins := []string{"http://localhost:3000", "https://app.4yaar.com"}
	upgrader := NewWebSocketUpgrader(allowedOrigins)

	tests := []struct {
		name       string
		origin     string
		wantResult bool
	}{
		{
			name:       "exact match allowed origin",
			origin:     "http://localhost:3000",
			wantResult: true,
		},
		{
			name:       "https allowed origin",
			origin:     "https://app.4yaar.com",
			wantResult: true,
		},
		{
			name:       "disallowed evil origin",
			origin:     "https://evil-attacker.com",
			wantResult: false,
		},
		{
			name:       "empty origin header (e.g. backend/native client)",
			origin:     "",
			wantResult: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/ws", nil)
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			result := upgrader.CheckOrigin(req)
			if result != tc.wantResult {
				t.Errorf("CheckOrigin(%q) = %v, want %v", tc.origin, result, tc.wantResult)
			}
		})
	}
}

func TestWebSocketUpgrader_WildcardOrigin(t *testing.T) {
	upgrader := NewWebSocketUpgrader([]string{"*"})
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Origin", "https://any-domain.org")
	if !upgrader.CheckOrigin(req) {
		t.Error("expected wildcard '*' to permit any origin")
	}
}

func TestExtractWebSocketToken(t *testing.T) {
	tests := []struct {
		name       string
		authHeader string
		queryParam string
		wantToken  string
	}{
		{
			name:       "extracted from Authorization Bearer header",
			authHeader: "Bearer access-token-abc-123",
			queryParam: "",
			wantToken:  "access-token-abc-123",
		},
		{
			name:       "extracted from lowercase bearer header",
			authHeader: "bearer access-token-abc-123",
			queryParam: "",
			wantToken:  "access-token-abc-123",
		},
		{
			name:       "falls back to query parameter when header is absent",
			authHeader: "",
			queryParam: "query-token-xyz-789",
			wantToken:  "query-token-xyz-789",
		},
		{
			name:       "header takes precedence over query parameter",
			authHeader: "Bearer header-token",
			queryParam: "query-token",
			wantToken:  "header-token",
		},
		{
			name:       "neither header nor query param present",
			authHeader: "",
			queryParam: "",
			wantToken:  "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			url := "/ws"
			if tc.queryParam != "" {
				url += "?token=" + tc.queryParam
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}

			token := ExtractWebSocketToken(req)
			if token != tc.wantToken {
				t.Errorf("ExtractWebSocketToken() = %q, want %q", token, tc.wantToken)
			}
		})
	}
}
