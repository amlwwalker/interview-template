package llm

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/alex/crudapi/internal/httpx"
)

// Handler serves the catalogue over HTTP. It knows nothing about Postgres —
// only about the Store interface, which is what lets its tests run with no
// database.
type Handler struct {
	store  Store
	logger *slog.Logger
}

// NewHandler builds a handler around any Store implementation.
func NewHandler(store Store, logger *slog.Logger) *Handler {
	return &Handler{store: store, logger: logger}
}

// Routes returns this resource's subrouter, ready to be mounted.
//
// Read-only by design: the catalogue is managed by migration, so the write
// verbs are registered purely to answer 405 rather than falling through to a
// 404, which would wrongly suggest the endpoint does not exist.
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.list)

	notAllowed := func(w http.ResponseWriter, _ *http.Request) {
		httpx.Error(w, http.StatusMethodNotAllowed, "the llm catalogue is read-only")
	}
	r.Post("/", notAllowed)
	r.Put("/", notAllowed)
	r.Patch("/", notAllowed)
	r.Delete("/", notAllowed)

	return r
}

// list returns the LLMs currently on offer, in display order.
//
// The response carries slug (what clients select by) and name (what a person
// reads). It does not carry provider_key: that is an internal binding to
// compiled code, and exposing it would invite selecting by it.
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	llms, err := h.store.List(r.Context())
	if err != nil {
		// Log the real error in full; return a bare message. Nothing about the
		// database reaches the client.
		h.logger.Error("list llms failed",
			"error", err,
			"request_id", r.Header.Get("X-Request-Id"),
		)
		httpx.Error(w, http.StatusServiceUnavailable, "llm catalogue unavailable")
		return
	}

	httpx.JSON(w, http.StatusOK, llms)
}
