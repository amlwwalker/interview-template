package record

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// fakeStore is an in-memory Store. Because the handlers depend on the Store
// interface, these tests need no database and run in milliseconds.
type fakeStore struct {
	records map[int64]Record
	nextID  int64
	err     error // when set, every method returns it
}

func newFakeStore() *fakeStore {
	return &fakeStore{records: map[int64]Record{}, nextID: 1}
}

func (f *fakeStore) List(context.Context) ([]Record, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := make([]Record, 0, len(f.records))
	for _, r := range f.records {
		out = append(out, r)
	}
	return out, nil
}

func (f *fakeStore) Get(_ context.Context, id int64) (Record, error) {
	if f.err != nil {
		return Record{}, f.err
	}
	r, ok := f.records[id]
	if !ok {
		return Record{}, ErrNotFound
	}
	return r, nil
}

func (f *fakeStore) Create(_ context.Context, p CreateParams) (Record, error) {
	if f.err != nil {
		return Record{}, f.err
	}
	r := Record{
		ID:          f.nextID,
		Name:        p.Name,
		Description: p.Description,
		Active:      p.Active,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	f.records[r.ID] = r
	f.nextID++
	return r, nil
}

// Replace mirrors the SQL: every mutable field is overwritten unconditionally.
func (f *fakeStore) Replace(_ context.Context, id int64, p ReplaceParams) (Record, error) {
	if f.err != nil {
		return Record{}, f.err
	}
	existing, ok := f.records[id]
	if !ok {
		return Record{}, ErrNotFound
	}
	replaced := Record{
		ID:          existing.ID,
		Name:        p.Name,
		Description: p.Description,
		Active:      p.Active,
		CreatedAt:   existing.CreatedAt,
		UpdatedAt:   time.Now(),
	}
	f.records[id] = replaced
	return replaced, nil
}

// Update mirrors the COALESCE: only supplied fields change.
func (f *fakeStore) Update(_ context.Context, id int64, p UpdateParams) (Record, error) {
	if f.err != nil {
		return Record{}, f.err
	}
	r, ok := f.records[id]
	if !ok {
		return Record{}, ErrNotFound
	}
	if p.Name != nil {
		r.Name = *p.Name
	}
	if p.Description != nil {
		r.Description = *p.Description
	}
	if p.Active != nil {
		r.Active = *p.Active
	}
	r.UpdatedAt = time.Now()
	f.records[id] = r
	return r, nil
}

func (f *fakeStore) Delete(_ context.Context, id int64) error {
	if f.err != nil {
		return f.err
	}
	if _, ok := f.records[id]; !ok {
		return ErrNotFound
	}
	delete(f.records, id)
	return nil
}

// newTestServer returns a server mounting the record routes, plus the fake
// store behind it.
func newTestServer(t *testing.T) (*httptest.Server, *fakeStore) {
	t.Helper()

	store := newFakeStore()
	// Discard logs so a deliberate 500 does not spam the test output.
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := NewHandler(store, logger)

	srv := httptest.NewServer(handler.Routes())
	t.Cleanup(srv.Close)

	return srv, store
}

func do(t *testing.T, srv *httptest.Server, method, path, body string) *http.Response {
	t.Helper()

	var reader io.Reader
	if body != "" {
		reader = bytes.NewBufferString(body)
	}

	req, err := http.NewRequest(method, srv.URL+path, reader)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })

	return resp
}

func decodeRecord(t *testing.T, resp *http.Response) Record {
	t.Helper()

	var out Record
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode record: %v", err)
	}
	return out
}

// seed inserts a fully-populated record so replace/update tests have something
// with non-zero values in every field.
func seed(t *testing.T, store *fakeStore) Record {
	t.Helper()

	rec, err := store.Create(context.Background(), CreateParams{
		Name:        "seeded",
		Description: "seeded description",
		Active:      true,
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	return rec
}

// --- CREATE ------------------------------------------------------------------

// POST stores every field, trims the name and returns 201 with a Location header.
func TestCreate(t *testing.T) {
	srv, _ := newTestServer(t)

	resp := do(t, srv, http.MethodPost, "/", `{"name":"  alpha  ","description":"first","active":true}`)

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	got := decodeRecord(t, resp)
	if got.Name != "alpha" {
		t.Errorf("name = %q, want %q (should be trimmed)", got.Name, "alpha")
	}
	if !got.Active {
		t.Error("active = false, want true")
	}
	if got.ID == 0 {
		t.Error("expected a generated id")
	}
	if loc := resp.Header.Get("Location"); loc != "/1" {
		t.Errorf("Location = %q, want %q", loc, "/1")
	}
}

// Bad bodies are rejected: 422 for invalid values, 400 for malformed JSON.
func TestCreateValidation(t *testing.T) {
	tests := []struct {
		name string
		body string
		want int
	}{
		{"empty name", `{"name":""}`, http.StatusUnprocessableEntity},
		{"whitespace name", `{"name":"   "}`, http.StatusUnprocessableEntity},
		{"missing name", `{"description":"orphan"}`, http.StatusUnprocessableEntity},
		{"malformed json", `{"name":`, http.StatusBadRequest},
		{"unknown field", `{"name":"x","colour":"red"}`, http.StatusBadRequest},
		{"two objects", `{"name":"a"}{"name":"b"}`, http.StatusBadRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv, _ := newTestServer(t)

			resp := do(t, srv, http.MethodPost, "/", tc.body)
			if resp.StatusCode != tc.want {
				t.Fatalf("status = %d, want %d", resp.StatusCode, tc.want)
			}
		})
	}
}

// --- READ --------------------------------------------------------------------

// An empty collection serialises as [] so the client needs no null check.
func TestListReturnsEmptyArrayNotNull(t *testing.T) {
	srv, _ := newTestServer(t)

	resp := do(t, srv, http.MethodGet, "/", "")

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	// `null` would force the frontend into defensive checks on every render.
	if got := bytes.TrimSpace(body); string(got) != "[]" {
		t.Errorf("body = %s, want []", got)
	}
}

// Reading a row that does not exist is a 404, not an error page.
func TestGetMissingIs404(t *testing.T) {
	srv, _ := newTestServer(t)

	resp := do(t, srv, http.MethodGet, "/999", "")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

// A non-numeric or non-positive id is rejected before touching the store.
func TestInvalidIDIs400(t *testing.T) {
	srv, _ := newTestServer(t)

	for _, path := range []string{"/abc", "/0", "/-1"} {
		resp := do(t, srv, http.MethodGet, path, "")
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("GET %s: status = %d, want %d", path, resp.StatusCode, http.StatusBadRequest)
		}
	}
}

// --- UPDATE: the PUT / PATCH distinction -------------------------------------

// This pair is the point of having both verbs. Same URL, same partial body,
// different outcome.

// PUT resets every field the body leaves out — the replacement semantic.
func TestPutReplacesOmittedFieldsWithZeroValues(t *testing.T) {
	srv, store := newTestServer(t)
	seed(t, store)

	// Only `name` is supplied. PUT means "the resource is now exactly this",
	// so description and active must be reset.
	resp := do(t, srv, http.MethodPut, "/1", `{"name":"replaced"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	got := decodeRecord(t, resp)
	if got.Name != "replaced" {
		t.Errorf("name = %q, want %q", got.Name, "replaced")
	}
	if got.Description != "" {
		t.Errorf("description = %q, want \"\" (PUT should have reset it)", got.Description)
	}
	if got.Active {
		t.Error("active = true, want false (PUT should have reset it)")
	}
}

// PATCH touches only the keys present in the body — the partial semantic.
func TestPatchLeavesOmittedFieldsAlone(t *testing.T) {
	srv, store := newTestServer(t)
	seed(t, store)

	// Byte-for-byte the same body as the PUT test above.
	resp := do(t, srv, http.MethodPatch, "/1", `{"name":"patched"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	got := decodeRecord(t, resp)
	if got.Name != "patched" {
		t.Errorf("name = %q, want %q", got.Name, "patched")
	}
	if got.Description != "seeded description" {
		t.Errorf("description = %q, want it untouched", got.Description)
	}
	if !got.Active {
		t.Error("active = false, want it untouched (true)")
	}
}

// PUT is idempotent: sending the same body twice leaves the same state.
func TestPutIsIdempotent(t *testing.T) {
	srv, store := newTestServer(t)
	seed(t, store)

	body := `{"name":"fixed","description":"fixed description","active":true}`

	first := decodeRecord(t, do(t, srv, http.MethodPut, "/1", body))
	second := decodeRecord(t, do(t, srv, http.MethodPut, "/1", body))

	if first.Name != second.Name ||
		first.Description != second.Description ||
		first.Active != second.Active {
		t.Errorf("PUT not idempotent: %+v then %+v", first, second)
	}
}

// PUT requires a full representation, so a missing name is a validation error
// rather than "leave it as it was".
func TestPutRequiresName(t *testing.T) {
	srv, store := newTestServer(t)
	seed(t, store)

	resp := do(t, srv, http.MethodPut, "/1", `{"description":"no name here"}`)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusUnprocessableEntity)
	}
}

// The reason UpdateParams uses pointers: clearing a boolean must be
// distinguishable from omitting it.
func TestPatchCanSetActiveToFalse(t *testing.T) {
	srv, store := newTestServer(t)
	seed(t, store) // seeded active = true

	resp := do(t, srv, http.MethodPatch, "/1", `{"active":false}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	got := decodeRecord(t, resp)
	if got.Active {
		t.Error("active = true, want false")
	}
	if got.Name != "seeded" {
		t.Errorf("name = %q, want it untouched", got.Name)
	}
}

// An empty PATCH body is a 422: there is no change to apply.
func TestPatchEmptyBodyIsRejected(t *testing.T) {
	srv, store := newTestServer(t)
	seed(t, store)

	resp := do(t, srv, http.MethodPatch, "/1", `{}`)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusUnprocessableEntity)
	}
}

// Both PUT and PATCH return 404 for a row that does not exist.
func TestUpdateMissingIs404(t *testing.T) {
	srv, _ := newTestServer(t)

	for _, method := range []string{http.MethodPut, http.MethodPatch} {
		resp := do(t, srv, method, "/999", `{"name":"x"}`)
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("%s: status = %d, want %d", method, resp.StatusCode, http.StatusNotFound)
		}
	}
}

// --- DELETE ------------------------------------------------------------------

// DELETE returns 204 and the row is genuinely gone afterwards.
func TestDeleteReturns204ThenGetIs404(t *testing.T) {
	srv, store := newTestServer(t)
	seed(t, store)

	if resp := do(t, srv, http.MethodDelete, "/1", ""); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}

	if resp := do(t, srv, http.MethodGet, "/1", ""); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("get-after-delete status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

// Deleting an already-deleted row reports 404 rather than pretending to succeed.
func TestDeleteMissingIs404(t *testing.T) {
	srv, _ := newTestServer(t)

	resp := do(t, srv, http.MethodDelete, "/999", "")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

// --- errors ------------------------------------------------------------------

// A store failure must surface as a 500 without leaking the underlying error.
func TestStoreFailureIs500AndDoesNotLeak(t *testing.T) {
	srv, store := newTestServer(t)
	store.err = errStub{}

	resp := do(t, srv, http.MethodGet, "/", "")
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusInternalServerError)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if bytes.Contains(body, []byte("connection refused on 127.0.0.1")) {
		t.Errorf("response leaked internal error: %s", body)
	}
}

type errStub struct{}

func (errStub) Error() string { return "connection refused on 127.0.0.1" }
