package storage

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/JD1359/notification-system-go/internal/models"
)

type Postgres struct {
	pool *pgxpool.Pool
}

func NewPostgres(ctx context.Context, url string) (*Postgres, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}
	return &Postgres{pool: pool}, nil
}

func (p *Postgres) Ping(ctx context.Context) error { return p.pool.Ping(ctx) }
func (p *Postgres) Close()                          { p.pool.Close() }

func (p *Postgres) Insert(ctx context.Context, n *models.Notification) error {
	_, err := p.pool.Exec(ctx, `
		INSERT INTO notifications (id, channel, to_address, subject, body, metadata, status, attempts, queued_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (id) DO NOTHING
	`, n.ID, n.Channel, n.To, n.Subject, n.Body, n.Metadata, n.Status, n.Attempts, n.QueuedAt, n.UpdatedAt)
	return err
}

func (p *Postgres) UpdateStatus(ctx context.Context, n *models.Notification) error {
	_, err := p.pool.Exec(ctx, `
		UPDATE notifications
		SET status = $2, attempts = $3, last_error = $4, updated_at = $5
		WHERE id = $1
	`, n.ID, n.Status, n.Attempts, n.LastError, n.UpdatedAt)
	return err
}

func (p *Postgres) Get(ctx context.Context, id string) (*models.Notification, error) {
	row := p.pool.QueryRow(ctx, `
		SELECT id, channel, to_address, subject, body, metadata, status, attempts, queued_at, updated_at, COALESCE(last_error,'')
		FROM notifications WHERE id = $1
	`, id)
	var n models.Notification
	if err := row.Scan(&n.ID, &n.Channel, &n.To, &n.Subject, &n.Body, &n.Metadata, &n.Status, &n.Attempts, &n.QueuedAt, &n.UpdatedAt, &n.LastError); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &n, nil
}

func (p *Postgres) AppendLog(ctx context.Context, l *models.DeliveryLog) error {
	_, err := p.pool.Exec(ctx, `
		INSERT INTO delivery_logs (notification_id, attempt, status, error, at)
		VALUES ($1,$2,$3,$4,$5)
	`, l.NotificationID, l.Attempt, l.Status, l.Error, l.At)
	return err
}
