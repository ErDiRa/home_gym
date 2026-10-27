package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/erdira/homegym/backend"
	"github.com/erdira/homegym/backend/common/log"
	"github.com/erdira/homegym/backend/modules/whoop/adapters/whoopapi"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

func run() error {
	logger, err := log.Setup(os.Getenv("LOG_LEVEL"), os.Getenv("LOG_FORMAT"))
	if err != nil {
		return err
	}

	// Cancelled on Ctrl+C or SIGTERM (docker stop), which triggers the graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	dbSource := os.Getenv("DB_SOURCE")
	if dbSource == "" {
		return errors.New("DB_SOURCE is not set")
	}
	pool, err := backend.NewPool(ctx, dbSource)
	if err != nil {
		return err
	}
	defer pool.Close()

	svc, err := backend.New(ctx, pool, logger, backend.Config{
		JWTSecret:    os.Getenv("JWT_SECRET"),
		CookieSecure: os.Getenv("COOKIE_SECURE") == "true",
		CORSOrigins:  splitEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173"),
		StaticDir:    os.Getenv("STATIC_DIR"),
		Whoop: whoopapi.Config{
			ClientID:     os.Getenv("WHOOP_CLIENT_ID"),
			ClientSecret: os.Getenv("WHOOP_CLIENT_SECRET"),
			RedirectURL:  os.Getenv("WHOOP_REDIRECT_URL"),
		},
	})
	if err != nil {
		return err
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return svc.Run(ctx, ":"+port)
}

// splitEnv reads a comma-separated list from the environment, using fallback when unset.
func splitEnv(key, fallback string) []string {
	v := os.Getenv(key)
	if v == "" {
		v = fallback
	}
	return strings.Split(v, ",")
}
