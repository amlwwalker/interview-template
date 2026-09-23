package llm

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestHandler(store Store) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewHandler(store, logger).Routes()
}

func TestListLLMsReturns200AndAJSONArray(t *testing.T) {
	h := newTestHandler(newFakeStore(
		LLM{ID: 1, Slug: "echo-mock", Name: "Echo (mock)", ProviderKey: KeyEcho, Model: "echo-v1", Enabled: true},
	))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var got []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v (body %q)", err, rec.Body.String())
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0]["slug"] != "echo-mock" || got[0]["name"] != "Echo (mock)" {
		t.Errorf("unexpected body: %+v", got[0])
	}
}

// An empty catalogue must serialise as [], not null, or every client has to
// null-check before iterating.
func TestListLLMsSerialisesEmptyCatalogueAsArray(t *testing.T) {
	h := newTestHandler(newFakeStore())

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if body := strings.TrimSpace(rec.Body.String()); body != "[]" {
		t.Errorf("body = %q, want %q", body, "[]")
	}
}

// provider_key is an internal binding. If it reaches a client, someone will
// start selecting by it, and the slug/name split stops meaning anything.
func TestListLLMsNeverExposesTheProviderKey(t *testing.T) {
	h := newTestHandler(newFakeStore(
		LLM{ID: 1, Slug: "echo-mock", Name: "Echo (mock)", ProviderKey: KeyEcho, Model: "echo-v1", Enabled: true},
	))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	body := rec.Body.String()
	for _, forbidden := range []string{"providerKey", "provider_key", `"echo"`} {
		if strings.Contains(body, forbidden) {
			t.Errorf("response leaks %q: %s", forbidden, body)
		}
	}
}

// A store failure is our problem, not the client's, and the underlying error
// must not reach them.
func TestListLLMsReturns503AndLeaksNothingWhenTheStoreFails(t *testing.T) {
	store := newFakeStore()
	store.err = errors.New("pq: relation \"llms\" does not exist: connection refused")

	h := newTestHandler(store)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}

	body := rec.Body.String()
	for _, leak := range []string{"pq:", "relation", "connection refused"} {
		if strings.Contains(body, leak) {
			t.Errorf("response leaks internal detail %q: %s", leak, body)
		}
	}
}

// The catalogue is read-only in this ticket. Writing to it must be refused
// explicitly rather than falling through to a 404, which would suggest the
// endpoint does not exist.
func TestWritesToTheCatalogueAreRejected(t *testing.T) {
	h := newTestHandler(newFakeStore())

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(method, "/", nil))

			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("status = %d, want 405", rec.Code)
			}
		})
	}
}
