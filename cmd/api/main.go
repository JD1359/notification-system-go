package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/JD1359/notification-system-go/internal/api"
	"github.com/JD1359/notification-system-go/internal/observability"
	"github.com/JD1359/notification-system-go/internal/queue"
	"github.com/JD1359/notification-system-go/internal/storage"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	addr := envOr("HTTP_ADDR", ":8080")
	redisURL := envOr("REDIS_URL", "redis://localhost:6379")
	pgURL := envOr("POSTGRES_URL", "postgres://app:app@localhost:5432/notifications?sslmode=disable")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store, err := storage.NewPostgres(ctx, pgURL)
	if err != nil {
		log.Fatal().Err(err).Msg("postgres connect")
	}
	defer store.Close()

	q, err := queue.NewRedisStream(ctx, redisURL, "notifications")
	if err != nil {
		log.Fatal().Err(err).Msg("redis connect")
	}
	defer q.Close()

	h := api.NewHandlers(store, q)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(observability.RequestLogger())
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(15 * time.Second))

	r.Get("/healthz", api.Healthz)
	r.Get("/readyz", api.Readyz(store, q))
	r.Handle("/metrics", observability.PrometheusHandler())

	r.Route("/v1", func(r chi.Router) {
		r.Post("/notifications", h.Enqueue)
		r.Get("/notifications/{id}", h.GetByID)
	})

	srv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Info().Str("addr", addr).Msg("api listening")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("api serve")
		}
	}()

	// Graceful shutdown
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Info().Msg("shutdown signal received")

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		log.Error().Err(err).Msg("api shutdown")
	}
	log.Info().Msg("api stopped")
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
