CREATE TYPE notification_channel AS ENUM ('sms', 'email', 'push');

CREATE TYPE notification_priority AS ENUM ('high', 'normal', 'low');

CREATE TYPE notification_status AS ENUM (
    'pending',
    'queued',
    'processing',
    'sent',
    'failed',
    'cancelled'
);

CREATE TABLE notification_batches (
    id UUID PRIMARY KEY,
    idempotency_key TEXT UNIQUE,
    total_count INTEGER NOT NULL CHECK (total_count > 0 AND total_count <= 1000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE message_templates (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    channel notification_channel NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE notifications (
    id UUID PRIMARY KEY,
    batch_id UUID REFERENCES notification_batches(id) ON DELETE SET NULL,
    template_id UUID REFERENCES message_templates(id) ON DELETE SET NULL,
    recipient TEXT NOT NULL,
    channel notification_channel NOT NULL,
    content TEXT NOT NULL,
    priority notification_priority NOT NULL DEFAULT 'normal',
    status notification_status NOT NULL DEFAULT 'pending',
    attempt_count INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
    max_attempts INTEGER NOT NULL DEFAULT 5 CHECK (max_attempts > 0),
    scheduled_at TIMESTAMPTZ,
    next_attempt_at TIMESTAMPTZ,
    provider_message_id TEXT,
    last_error TEXT,
    cancelled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notifications_batch_id ON notifications(batch_id);
CREATE INDEX idx_notifications_status ON notifications(status);
CREATE INDEX idx_notifications_channel ON notifications(channel);
CREATE INDEX idx_notifications_created_at ON notifications(created_at DESC);
CREATE INDEX idx_notifications_next_attempt_at ON notifications(next_attempt_at)
    WHERE status = 'pending';
CREATE INDEX idx_notifications_scheduled_at
    ON notifications(scheduled_at)
    WHERE scheduled_at IS NOT NULL;
CREATE INDEX idx_notifications_template_id
    ON notifications(template_id)
    WHERE template_id IS NOT NULL;

CREATE TABLE notification_attempts (
    id UUID PRIMARY KEY,
    notification_id UUID NOT NULL REFERENCES notifications(id) ON DELETE CASCADE,
    attempt_number INTEGER NOT NULL CHECK (attempt_number > 0),
    response_status_code INTEGER,
    response_body TEXT,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (notification_id, attempt_number)
);

CREATE INDEX idx_notification_attempts_notification_id
    ON notification_attempts(notification_id, created_at DESC);

CREATE TABLE idempotency_keys (
    id UUID PRIMARY KEY,
    idempotency_key TEXT NOT NULL UNIQUE,
    batch_id UUID REFERENCES notification_batches(id) ON DELETE SET NULL,
    notification_id UUID REFERENCES notifications(id) ON DELETE SET NULL,
    request_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE outbox_events (
    id UUID PRIMARY KEY,
    aggregate_type TEXT NOT NULL,
    aggregate_id UUID NOT NULL,
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    published_at TIMESTAMPTZ,
    available_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_outbox_events_ready
    ON outbox_events(available_at, created_at)
    WHERE published_at IS NULL;

COMMENT ON TABLE outbox_events IS
'Stores broker publication events in the same transaction as domain writes to avoid the PostgreSQL/RabbitMQ dual-write gap.';
