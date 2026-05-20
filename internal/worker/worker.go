package worker

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/JD1359/notification-system-go/internal/channels"
	"github.com/JD1359/notification-system-go/internal/models"
	"github.com/JD1359/notification-system-go/internal/queue"
	"github.com/JD1359/notification-system-go/internal/storage"
)

const (
	maxAttempts = 3
	baseBackoff = 5 * time.Second
)

type Pool struct {
	q        *queue.RedisStream
	store    *storage.Postgres
	channels *channels.Registry
	size     int
}

func NewPool(q *queue.RedisStream, s *storage.Postgres, c *channels.Registry, size int) *Pool {
	return &Pool{q: q, store: s, channels: c, size: size}
}

func (p *Pool) Run(ctx context.Context) {
	var wg sync.WaitGroup
	for i := 0; i < p.size; i++ {
		wg.Add(1)
		consumer := fmt.Sprintf("worker-%s-%d", uuid.New().String()[:8], i)
		go func() {
			defer wg.Done()
			p.runWorker(ctx, consumer)
		}()
	}
	wg.Wait()
}

func (p *Pool) runWorker(ctx context.Context, consumer string) {
	log.Info().Str("consumer", consumer).Msg("worker started")
	for {
		select {
		case <-ctx.Done():
			log.Info().Str("consumer", consumer).Msg("worker stopping")
			return
		default:
		}
		msgs, err := p.q.Read(ctx, consumer, 5*time.Second)
		if err != nil {
			log.Error().Err(err).Str("consumer", consumer).Msg("read failed")
			time.Sleep(time.Second)
			continue
		}
		for _, m := range msgs {
			p.handle(ctx, m)
		}
	}
}

func (p *Pool) handle(ctx context.Context, m queue.Message) {
	n := m.Payload
	n.Status = models.StatusInFlight
	n.Attempts++
	n.UpdatedAt = time.Now().UTC()

	if err := p.store.UpdateStatus(ctx, n); err != nil {
		log.Error().Err(err).Str("id", n.ID).Msg("status update")
	}

	adapter, ok := p.channels.Get(string(n.Channel))
	if !ok {
		log.Error().Str("channel", string(n.Channel)).Msg("no adapter for channel")
		p.fail(ctx, n, fmt.Errorf("no adapter for channel %s", n.Channel))
		_ = p.q.Ack(ctx, m.ID)
		return
	}

	err := adapter.Send(ctx, n)
	if err == nil {
		n.Status = models.StatusDelivered
		n.UpdatedAt = time.Now().UTC()
		_ = p.store.UpdateStatus(ctx, n)
		_ = p.store.AppendLog(ctx, &models.DeliveryLog{NotificationID: n.ID, Attempt: n.Attempts, Status: models.StatusDelivered, At: time.Now().UTC()})
		_ = p.q.Ack(ctx, m.ID)
		log.Info().Str("id", n.ID).Str("channel", string(n.Channel)).Msg("delivered")
		return
	}

	log.Warn().Err(err).Str("id", n.ID).Int("attempt", n.Attempts).Msg("delivery failed")
	_ = p.store.AppendLog(ctx, &models.DeliveryLog{NotificationID: n.ID, Attempt: n.Attempts, Status: models.StatusFailed, Error: err.Error(), At: time.Now().UTC()})

	if n.Attempts >= maxAttempts {
		p.fail(ctx, n, err)
		_ = p.q.Ack(ctx, m.ID)
		return
	}

	// Exponential backoff, then re-enqueue
	backoff := time.Duration(math.Pow(5, float64(n.Attempts))) * time.Second
	if backoff < baseBackoff {
		backoff = baseBackoff
	}
	time.AfterFunc(backoff, func() {
		bg := context.Background()
		_ = p.q.Enqueue(bg, n)
	})
	_ = p.q.Ack(ctx, m.ID)
}

func (p *Pool) fail(ctx context.Context, n *models.Notification, err error) {
	n.Status = models.StatusDeadLet
	n.LastError = err.Error()
	n.UpdatedAt = time.Now().UTC()
	_ = p.store.UpdateStatus(ctx, n)
	_ = p.q.DeadLetter(ctx, n)
	log.Error().Str("id", n.ID).Msg("dead-lettered")
}
