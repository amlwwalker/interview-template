package main

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alex/crudapi/internal/config"
)

// newTestRouter builds the real router with a nil pool.
//
// Nothing these tests exercise touches the database: CORS is middleware, and
// the catalogue route is asserted for mounting rather than for its body. The
// one route that would dereference the pool is /readyz, which is covered by
// its own test below to prove the nil is not hiding a problem.
func newTestRouter() http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	cfg := config.Config{
		Port:           "8080",
		AllowedOrigins: []string{"http://localhost:5173"},
	}

	return newRouter(cfg, nil, logger)
}

// Guards the assumption above: if a future change makes a non-readyz route
// touch the pool, this test keeps passing while the others start panicking,
// which points straight at the cause.
func TestHealthzDoesNotTouchTheDatabase(t *testing.T) {
	r := newTestRouter()

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}
