package queue

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/JD1359/notification-system-go/internal/models"
)

// RedisStream wraps a Redis Streams-backed queue using consumer groups for
// at-least-once delivery with parallel work distribution.
type RedisStream struct {
	client *redis.Client
	stream string
	group  string
}

func NewRedisStream(ctx context.Context, url, stream string) (*RedisStream, error) {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}
	c := redis.NewClient(opt)
	if err := c.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	q := &RedisStream{client: c, stream: stream, group: "workers"}
	// Best-effort group creation; BUSYGROUP is fine.
	_ = c.XGroupCreateMkStream(ctx, stream, q.group, "$").Err()
	return q, nil
}

func (q *RedisStream) Ping(ctx context.Context) error {
	return q.client.Ping(ctx).Err()
}

func (q *RedisStream) Close() error {
	return q.client.Close()
}

func (q *RedisStream) Enqueue(ctx context.Context, n *models.Notification) error {
	payload, err := json.Marshal(n)
	if err != nil {
		return err
	}
	return q.client.XAdd(ctx, &redis.XAddArgs{
		Stream: q.stream,
		Values: map[string]interface{}{"payload": payload},
	}).Err()
}

type Message struct {
	ID      string
	Payload *models.Notification
}

// Read blocks for up to `block` waiting for messages addressed to `consumer`.
func (q *RedisStream) Read(ctx context.Context, consumer string, block time.Duration) ([]Message, error) {
	res, err := q.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    q.group,
		Consumer: consumer,
		Streams:  []string{q.stream, ">"},
		Count:    16,
		Block:    block,
	}).Result()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, nil
	}
	out := make([]Message, 0, len(res[0].Messages))
	for _, m := range res[0].Messages {
		raw, ok := m.Values["payload"].(string)
		if !ok {
			continue
		}
		var n models.Notification
		if err := json.Unmarshal([]byte(raw), &n); err != nil {
			continue
		}
		out = append(out, Message{ID: m.ID, Payload: &n})
	}
	return out, nil
}

func (q *RedisStream) Ack(ctx context.Context, id string) error {
	return q.client.XAck(ctx, q.stream, q.group, id).Err()
}

// DeadLetter pushes a permanently-failed message onto a separate stream for
// manual inspection.
func (q *RedisStream) DeadLetter(ctx context.Context, n *models.Notification) error {
	payload, err := json.Marshal(n)
	if err != nil {
		return err
	}
	return q.client.XAdd(ctx, &redis.XAddArgs{
		Stream: q.stream + ":dlq",
		Values: map[string]interface{}{"payload": payload},
	}).Err()
}
