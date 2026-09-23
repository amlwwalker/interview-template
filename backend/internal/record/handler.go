package record

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/alex/crudapi/internal/httpx"
)

// Handler turns HTTP requests into Store calls. It knows nothing about
// Postgres — only about the Store interface.
type Handler struct {
	store  Store
	logger *slog.Logger
}

// NewHandler builds a handler around any Store implementation.
func NewHandler(store Store, logger *slog.Logger) *Handler {
	return &Handler{store: store, logger: logger}
}

// Routes returns this resource's subrouter, ready to be mounted.
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.list)          // READ   (collection)
	r.Post("/", h.create)       // CREATE
	r.Get("/{id}", h.get)       // READ   (single)
	r.Put("/{id}", h.put)       // UPDATE (full replace)
	r.Patch("/{id}", h.patch)   // UPDATE (partial)
	r.Delete("/{id}", h.delete) // DELETE

	return r
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	records, err := h.store.List(r.Context())
	if err != nil {
		h.serverError(w, r, "list records", err)
		return
	}

	httpx.JSON(w, http.StatusOK, records)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	rec, err := h.store.Get(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, "record not found")
		return
	}
	if err != nil {
		h.serverError(w, r, "get record", err)
		return
	}

	httpx.JSON(w, http.StatusOK, rec)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var params CreateParams
	if !decodeJSON(w, r, &params) {
		return
	}

	if problems := params.Validate(); problems != nil {
		httpx.ValidationError(w, problems)
		return
	}

	rec, err := h.store.Create(r.Context(), params)
	if err != nil {
		h.serverError(w, r, "create record", err)
		return
	}

	// 201 plus a Location header pointing at the new resource. r.URL.Path is the
	// full request path even inside a mounted subrouter, and it may or may not
	// carry a trailing slash, so normalise before appending the id.
	base := strings.TrimSuffix(r.URL.Path, "/")
	w.Header().Set("Location", base+"/"+strconv.FormatInt(rec.ID, 10))
	httpx.JSON(w, http.StatusCreated, rec)
}

// put replaces the whole resource. Every mutable field is overwritten with what
// the body contained, so omitting a field resets it.
func (h *Handler) put(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	var params ReplaceParams
	if !decodeJSON(w, r, &params) {
		return
	}

	if problems := params.Validate(); problems != nil {
		httpx.ValidationError(w, problems)
		return
	}

	rec, err := h.store.Replace(r.Context(), id, params)
	if errors.Is(err, ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, "record not found")
		return
	}
	if err != nil {
		h.serverError(w, r, "replace record", err)
		return
	}

	httpx.JSON(w, http.StatusOK, rec)
}

// patch applies only the fields present in the body.
func (h *Handler) patch(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	var params UpdateParams
	if !decodeJSON(w, r, &params) {
		return
	}

	if problems := params.Validate(); problems != nil {
		httpx.ValidationError(w, problems)
		return
	}

	rec, err := h.store.Update(r.Context(), id, params)
	if errors.Is(err, ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, "record not found")
		return
	}
	if err != nil {
		h.serverError(w, r, "update record", err)
		return
	}

	httpx.JSON(w, http.StatusOK, rec)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	err := h.store.Delete(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, "record not found")
		return
	}
	if err != nil {
		h.serverError(w, r, "delete record", err)
		return
	}

	httpx.NoContent(w)
}

// parseID pulls {id} off the URL, writing a 400 and returning false if it is
// not a positive integer.
func (h *Handler) parseID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	raw := chi.URLParam(r, "id")

	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id < 1 {
		httpx.Error(w, http.StatusBadRequest, "id must be a positive integer")
		return 0, false
	}

	return id, true
}

// serverError logs the real cause and returns a deliberately vague message —
// internal errors are for the operator, not the client.
func (h *Handler) serverError(w http.ResponseWriter, r *http.Request, op string, err error) {
	h.logger.Error("request failed",
		"op", op,
		"method", r.Method,
		"path", r.URL.Path,
		"error", err,
	)
	httpx.Error(w, http.StatusInternalServerError, "internal server error")
}

// decodeJSON reads exactly one JSON object into dst, rejecting unknown fields
// and trailing data. Returns false if it already wrote an error response.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	// Cap the body so a malicious client cannot exhaust memory.
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		httpx.Error(w, http.StatusBadRequest, "request body must be valid JSON: "+err.Error())
		return false
	}

	// A second value in the stream means the client sent something we did not
	// expect, e.g. two concatenated objects.
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		httpx.Error(w, http.StatusBadRequest, "request body must contain a single JSON object")
		return false
	}

	return true
}
