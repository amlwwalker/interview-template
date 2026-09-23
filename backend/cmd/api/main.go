package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/alex/crudapi/internal/config"
	"github.com/alex/crudapi/internal/database"
	"github.com/alex/crudapi/internal/httpx"
	"github.com/alex/crudapi/internal/record"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		logger.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	// A missing .env is fine — the environment may already be populated.
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	logger.Info("connected to postgres")

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           newRouter(cfg, pool, logger),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Run the server in the background so main can wait on the signal context.
	serverErr := make(chan error, 1)
	go func() {
		logger.Info("listening",
			"addr", srv.Addr,
			"allowed_origins", cfg.AllowedOrigins,
		)

		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		return err

	case <-ctx.Done():
		logger.Info("shutdown signal received, draining connections")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			return err
		}

		logger.Info("shutdown complete")
		return nil
	}
}

func newRouter(cfg config.Config, pool *pgxpool.Pool, logger *slog.Logger) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	// CORS must be registered before the routes it protects. chi/cors answers
	// the browser's preflight OPTIONS request itself, which is the piece people
	// usually miss when hand-rolling this.
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: cfg.AllowedOrigins,
		// Every verb the API serves must be listed here, or the browser rejects
		// the preflight and the request never reaches a handler. Forgetting PUT
		// or PATCH is the classic version of this bug: GET and POST keep working,
		// so the router looks fine.
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type", "X-Request-Id"},
		// Cross-origin JS can only read headers named here. Without Location the
		// browser receives the 201 header but cannot see it.
		ExposedHeaders: []string{"X-Request-Id", "Location"},
		// Leave false unless you move to cookie auth; `true` is incompatible
		// with an AllowedOrigins of "*".
		AllowCredentials: false,
		MaxAge:           300, // cache the preflight for 5 minutes
	}))

	// Liveness: is the process up? Deliberately does not touch the database,
	// so a database blip cannot cause a restart loop.
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Readiness: can we actually serve traffic? This one does check the database.
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := pool.Ping(ctx); err != nil {
			logger.Warn("readiness check failed", "error", err)
			httpx.Error(w, http.StatusServiceUnavailable, "database unavailable")
			return
		}

		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})

	recordHandler := record.NewHandler(record.NewPostgresStore(pool), logger)

	r.Route("/api/v1", func(r chi.Router) {
		r.Mount("/records", recordHandler.Routes())
	})

	r.NotFound(func(w http.ResponseWriter, _ *http.Request) {
		httpx.Error(w, http.StatusNotFound, "endpoint not found")
	})

	r.MethodNotAllowed(func(w http.ResponseWriter, _ *http.Request) {
		httpx.Error(w, http.StatusMethodNotAllowed, "method not allowed")
	})

	return r
}
