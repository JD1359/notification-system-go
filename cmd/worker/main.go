package main

import (
	"context"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"

	"github.com/rs/zerolog/log"

	"github.com/JD1359/notification-system-go/internal/channels"
	"github.com/JD1359/notification-system-go/internal/queue"
	"github.com/JD1359/notification-system-go/internal/storage"
	"github.com/JD1359/notification-system-go/internal/worker"
)

func main() {
	concurrency := envInt("WORKER_CONCURRENCY", 5)
	redisURL := envOr("REDIS_URL", "redis://localhost:6379")
	pgURL := envOr("POSTGRES_URL", "postgres://app:app@localhost:5432/notifications?sslmode=disable")

	ctx, cancel := context.WithCancel(context.Background())

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

	registry := channels.NewRegistry()
	registry.Register("email", channels.NewMockEmail())
	registry.Register("sms", channels.NewMockSMS())
	registry.Register("push", channels.NewMockPush())

	pool := worker.NewPool(q, store, registry, concurrency)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		pool.Run(ctx)
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Info().Msg("worker shutdown signal received")
	cancel()
	wg.Wait()
	log.Info().Msg("worker stopped")
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func envInt(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
