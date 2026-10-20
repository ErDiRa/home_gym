package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/erdira/homegym/backend/internal/auth"
	"github.com/erdira/homegym/backend/internal/handlers"
	"github.com/erdira/homegym/backend/internal/logging"
	"github.com/erdira/homegym/backend/internal/service/account"
	"github.com/erdira/homegym/backend/internal/service/exercise"
	"github.com/erdira/homegym/backend/internal/service/feedback"
	"github.com/erdira/homegym/backend/internal/service/lookup"
	"github.com/erdira/homegym/backend/internal/service/training"
	"github.com/erdira/homegym/backend/internal/service/workout"
	"github.com/erdira/homegym/backend/internal/web"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

func run() error {
	logger, err := logging.Setup(os.Getenv("LOG_LEVEL"), os.Getenv("LOG_FORMAT"))
	if err != nil {
		return err
	}

	// Cancelled on Ctrl+C or SIGTERM (docker stop), which triggers the graceful shutdown below.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	dbSource := os.Getenv("DB_SOURCE")
	if dbSource == "" {
		return errors.New("DB_SOURCE is not set")
	}

	dbpool, err := pgxpool.New(ctx, dbSource)
	if err != nil {
		return err
	}
	defer dbpool.Close()

	// pgxpool.New connects lazily; ping so a bad DB_SOURCE fails at startup, not on the first request.
	if err := dbpool.Ping(ctx); err != nil {
		return err
	}

	tokens, err := auth.NewTokenManager(os.Getenv("JWT_SECRET"))
	if err != nil {
		return err
	}

	// RequestID runs first so the logger and every error log see the same ID; Recoverer sits
	// inside the logger so a panic is still logged as a 500.
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(logging.Middleware(logger))
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   splitEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173"),
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type"},
		AllowCredentials: true, // the auth cookie must be sent cross-origin (e.g. without the Vite proxy)
		MaxAge:           300,
	}))

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	services := handlers.Services{
		Trainings: training.New(dbpool),
		Exercises: exercise.New(dbpool),
		Lookups:   lookup.New(dbpool),
		Feedback:  feedback.New(dbpool),
		Workouts:  workout.New(dbpool),
		Accounts:  account.New(dbpool),
	}
	handlers.New(services, tokens, os.Getenv("COOKIE_SECURE") == "true").Routes(r)
	// In production the built frontend is served next to the API (same origin, so the auth
	// cookie just works); in dev STATIC_DIR is unset and Vite serves it.
	if dir := os.Getenv("STATIC_DIR"); dir != "" {
		r.Handle("/*", web.SPA(dir))
		logger.Info("serving frontend", "dir", dir)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Buffered so the goroutine can exit even when nobody reads the error (after a shutdown).
	errCh := make(chan error, 1)
	go func() {
		logger.Info("listening", "port", port)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		// Stop accepting connections and give in-flight requests up to 10s to finish.
		logger.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

// splitEnv reads a comma-separated list from the environment, using fallback when unset.
func splitEnv(key, fallback string) []string {
	v := os.Getenv(key)
	if v == "" {
		v = fallback
	}
	return strings.Split(v, ",")
}
