package main

import (
	"context"
	"errors"
	"log"
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

	"github.com/stefankarlin/my-accessories/backend/internal/database"
	"github.com/stefankarlin/my-accessories/backend/internal/handlers"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
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

	if err := dbpool.Ping(ctx); err != nil {
		return err
	}

	queries := database.New(dbpool)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: splitEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173"),
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Content-Type", handlers.UserIDHeader},
		MaxAge:         300,
	}))

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handlers.New(queries).Routes(r)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("listening on :%s", port)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

func splitEnv(key, fallback string) []string {
	v := os.Getenv(key)
	if v == "" {
		v = fallback
	}
	return strings.Split(v, ",")
}
