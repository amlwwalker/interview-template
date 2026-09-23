package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The classic CORS bug is a verb missing from the allowed-methods list: the
// router looks fine because same-origin calls work, while the browser rejects
// the preflight. Assert the preflight response directly rather than leaving it
// to a human with a browser.
func TestPreflightForTheLLMCatalogueAllowsGET(t *testing.T) {
	r := newTestRouter()

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/llms", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", "GET")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK && rec.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want 200 or 204", rec.Code)
	}

	allowed := rec.Header().Get("Access-Control-Allow-Methods")
	if !strings.Contains(allowed, http.MethodGet) {
		t.Errorf("Access-Control-Allow-Methods = %q, want it to include GET", allowed)
	}

	if origin := rec.Header().Get("Access-Control-Allow-Origin"); origin == "" {
		t.Error("expected an Access-Control-Allow-Origin header on the preflight")
	}
}

// The route has to actually be mounted. Without this the handler tests pass
// while /api/v1/llms returns the router's 404.
func TestLLMCatalogueIsMountedUnderAPIV1(t *testing.T) {
	r := newTestRouter()

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/llms", nil))

	if rec.Code == http.StatusNotFound {
		t.Fatal("/api/v1/llms is not mounted")
	}
}
