CREATE TABLE IF NOT EXISTS notifications (
    id            TEXT PRIMARY KEY,
    channel       TEXT NOT NULL,
    to_address    TEXT NOT NULL,
    subject       TEXT,
    body          TEXT NOT NULL,
    metadata      JSONB,
    status        TEXT NOT NULL,
    attempts      INT  NOT NULL DEFAULT 0,
    last_error    TEXT,
    queued_at     TIMESTAMPTZ NOT NULL,
    updated_at    TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_notifications_status ON notifications(status);
CREATE INDEX IF NOT EXISTS idx_notifications_channel_status ON notifications(channel, status);
CREATE INDEX IF NOT EXISTS idx_notifications_queued_at ON notifications(queued_at DESC);

CREATE TABLE IF NOT EXISTS delivery_logs (
    id              BIGSERIAL PRIMARY KEY,
    notification_id TEXT NOT NULL REFERENCES notifications(id) ON DELETE CASCADE,
    attempt         INT  NOT NULL,
    status          TEXT NOT NULL,
    error           TEXT,
    at              TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_delivery_logs_notif ON delivery_logs(notification_id, attempt);
