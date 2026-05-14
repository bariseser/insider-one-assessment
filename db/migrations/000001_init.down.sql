DROP TABLE IF EXISTS outbox_events;
DROP TABLE IF EXISTS idempotency_keys;
DROP TABLE IF EXISTS notification_attempts;
DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS message_templates;
DROP TABLE IF EXISTS notification_batches;

DROP TYPE IF EXISTS notification_status;
DROP TYPE IF EXISTS notification_priority;
DROP TYPE IF EXISTS notification_channel;
