package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsAllowedLocalOrigin(t *testing.T) {
	tests := []struct {
		name   string
		origin string
		want   bool
	}{
		{name: "localhost vite", origin: "http://localhost:5173", want: true},
		{name: "loopback vite", origin: "http://127.0.0.1:5173", want: true},
		{name: "ipv6 loopback vite", origin: "http://[::1]:5173", want: true},
		{name: "wails app", origin: "wails://wails", want: true},
		{name: "empty", origin: "", want: false},
		{name: "wrong port", origin: "http://127.0.0.1:5174", want: false},
		{name: "remote host", origin: "https://example.com", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isAllowedLocalOrigin(tt.origin); got != tt.want {
				t.Fatalf("isAllowedLocalOrigin(%q) = %v, want %v", tt.origin, got, tt.want)
			}
		})
	}
}

func TestWithCORSPreflightAllowsLoopbackVite(t *testing.T) {
	state := &AppState{}
	req := httptest.NewRequest(http.MethodOptions, "/engine/preview", nil)
	req.Header.Set("Origin", "http://127.0.0.1:5173")
	rec := httptest.NewRecorder()

	state.withCORS(func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("preflight should not call wrapped handler")
	})(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://127.0.0.1:5173" {
		t.Fatalf("Access-Control-Allow-Origin = %q", got)
	}
}
