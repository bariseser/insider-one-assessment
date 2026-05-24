package repository

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"insider-one-assessment/internal/dto/request"
	"insider-one-assessment/internal/model"
	"insider-one-assessment/internal/tracing"
	postgresdb "insider-one-assessment/pkgs/postgres"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrIdempotencyConflict        = errors.New("idempotency key already exists with different payload")
	ErrNotificationNotFound       = errors.New("notification not found")
	ErrNotificationNotCancellable = errors.New("notification is not cancellable")
	ErrOutboxEventNotFound        = errors.New("outbox event not found")
	ErrNotificationNotReady       = errors.New("notification is not ready for processing")
	ErrTemplateNotFound           = errors.New("template not found")
	ErrTemplateAlreadyExists      = errors.New("template already exists")
)

type CreateNotificationsParams struct {
	IdempotencyKey string
	RequestHash    string
	Notifications  []request.NotificationItem
}

type ListNotificationsParams struct {
	Status   *model.NotificationStatus
	Channel  *model.NotificationChannel
	DateFrom *time.Time
	DateTo   *time.Time
	Limit    int
	Offset   int
}

type OutboxEvent struct {
	ID            string
	AggregateType string
	AggregateID   string
	EventType     string
	Payload       []byte
	AvailableAt   time.Time
	CreatedAt     time.Time
}

type NotificationAttemptResult struct {
	ResponseStatusCode *int
	ResponseBody       *string
	ErrorMessage       *string
}

type notificationRepository struct {
	db     *sql.DB
	gormDB *gorm.DB
}

func NewNotificationRepository(client postgresdb.IPostgresInstance) INotificationRepository {
	return &notificationRepository{
		db:     client.SQLDatabase(),
		gormDB: client.Database(),
	}
}

func HashCreateRequest(items []request.NotificationItem) (string, error) {
	payload, err := json.Marshal(items)
	if err != nil {
		return "", fmt.Errorf("marshal request hash payload: %w", err)
	}

	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func (r *notificationRepository) CreateNotification(ctx context.Context, idempotencyKey, requestHash string, item request.NotificationItem) (*model.Notification, bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, false, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	now := time.Now().UTC()
	if idempotencyKey != "" {
		notification, replayed, found, err := r.loadExistingNotificationIfPresent(ctx, tx, idempotencyKey, requestHash)
		if err != nil {
			return nil, false, err
		}
		if found {
			return &notification, replayed, nil
		}
	}

	notification, err := r.insertNotification(ctx, tx, nil, item, now)
	if err != nil {
		return nil, false, err
	}

	if idempotencyKey != "" {
		// UNIQUE constraint violation (23505) — RACE CONDITION!
		if err := r.insertNotificationIdempotencyKey(ctx, tx, idempotencyKey, requestHash, notification.ID); err != nil {
			if errors.Is(err, ErrIdempotencyConflict) {
				notification, replayed, err := r.loadExistingNotification(ctx, tx, idempotencyKey, requestHash)
				return &notification, replayed, err
			}
			return nil, false, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, false, fmt.Errorf("commit tx: %w", err)
	}

	return &notification, false, nil
}

func (r *notificationRepository) CreateNotifications(ctx context.Context, params CreateNotificationsParams) (model.NotificationBatch, []model.Notification, bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return model.NotificationBatch{}, nil, false, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	batchID := uuid.New()
	now := time.Now().UTC()

	if params.IdempotencyKey != "" {
		batch, notifications, replayed, found, err := r.loadExistingBatchIfPresent(ctx, tx, params.IdempotencyKey, params.RequestHash)
		if err != nil {
			return model.NotificationBatch{}, nil, false, err
		}
		if found {
			return *batch, notifications, replayed, nil
		}
	}

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO notification_batches (id, idempotency_key, total_count, created_at)
		 VALUES ($1, $2, $3, $4)`,
		batchID,
		nullableText(params.IdempotencyKey),
		len(params.Notifications),
		now,
	)
	if err != nil {
		return model.NotificationBatch{}, nil, false, fmt.Errorf("insert notification batch: %w", err)
	}

	if params.IdempotencyKey != "" {
		err = r.insertIdempotencyKey(ctx, tx, params.IdempotencyKey, params.RequestHash, batchID.String())
		if err != nil {
			if errors.Is(err, ErrIdempotencyConflict) {
				batch, notifications, replayed, loadErr := r.loadExistingBatch(ctx, tx, params.IdempotencyKey, params.RequestHash)
				if loadErr != nil {
					return model.NotificationBatch{}, nil, false, loadErr
				}
				return *batch, notifications, replayed, nil
			}
			return model.NotificationBatch{}, nil, false, err
		}
	}

	notifications := make([]model.Notification, 0, len(params.Notifications))
	for _, item := range params.Notifications {
		notification, err := r.insertNotification(ctx, tx, &batchID, item, now)
		if err != nil {
			return model.NotificationBatch{}, nil, false, err
		}
		notifications = append(notifications, notification)
	}

	if err := tx.Commit(); err != nil {
		return model.NotificationBatch{}, nil, false, fmt.Errorf("commit tx: %w", err)
	}

	return model.NotificationBatch{
		ID:             batchID,
		IdempotencyKey: optionalString(params.IdempotencyKey),
		TotalCount:     len(notifications),
		CreatedAt:      now,
	}, notifications, false, nil
}

func (r *notificationRepository) GetNotificationByID(ctx context.Context, id string) (*model.Notification, error) {
	notification, err := r.getNotificationByIDQuery(ctx, r.db.QueryRowContext, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotificationNotFound
		}
		return nil, fmt.Errorf("get notification by id: %w", err)
	}
	return &notification, nil
}

func (r *notificationRepository) GetBatchByID(ctx context.Context, id string) (*model.NotificationBatch, []model.Notification, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, nil, fmt.Errorf("begin readonly tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	batch, notifications, _, err := r.loadBatchWithNotifications(ctx, tx, id, false)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, ErrNotificationNotFound
		}
		return nil, nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("commit readonly tx: %w", err)
	}

	return batch, notifications, nil
}

func (r *notificationRepository) ListNotifications(ctx context.Context, params ListNotificationsParams) ([]model.Notification, int, error) {
	query := r.gormDB.WithContext(ctx).Model(&model.Notification{})
	if params.Status != nil {
		query = query.Where("status = ?", *params.Status)
	}
	if params.Channel != nil {
		query = query.Where("channel = ?", *params.Channel)
	}
	if params.DateFrom != nil {
		query = query.Where("created_at >= ?", *params.DateFrom)
	}
	if params.DateTo != nil {
		query = query.Where("created_at <= ?", *params.DateTo)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count notifications: %w", err)
	}

	notifications := make([]model.Notification, 0, params.Limit)
	if err := query.Order("created_at DESC").Limit(params.Limit).Offset(params.Offset).Find(&notifications).Error; err != nil {
		return nil, 0, fmt.Errorf("list notifications: %w", err)
	}

	return notifications, int(total), nil
}

func (r *notificationRepository) CancelNotification(ctx context.Context, id string, cancelledAt time.Time) (*model.Notification, error) {
	var notification model.Notification
	result := r.gormDB.WithContext(ctx).
		Model(&notification).
		Clauses(clause.Returning{}).
		Where("id = ? AND status IN ?", id, []model.NotificationStatus{model.StatusPending, model.StatusQueued}).
		Updates(map[string]any{
			"status":       model.StatusCancelled,
			"cancelled_at": cancelledAt,
			"updated_at":   cancelledAt,
		})

	if result.Error == nil && result.RowsAffected > 0 {
		return &notification, nil
	}
	if result.Error == nil && result.RowsAffected == 0 {
		current, getErr := r.GetNotificationByID(ctx, id)
		if getErr != nil {
			return nil, getErr
		}
		if current.Status != model.StatusPending && current.Status != model.StatusQueued {
			return nil, ErrNotificationNotCancellable
		}
		return nil, ErrNotificationNotFound
	}

	return nil, fmt.Errorf("cancel notification: %w", result.Error)
}

func (r *notificationRepository) ClaimDueOutboxEvents(ctx context.Context, limit int) ([]OutboxEvent, error) {
	now := time.Now().UTC()

	//atomic outbox claim: Race window: Commit ile MarkOutboxEventPublished
	rows, err := r.db.QueryContext(ctx, `
		WITH locked AS (
			SELECT id, aggregate_id FROM outbox_events
			WHERE published_at IS NULL AND available_at <= NOW()
			ORDER BY available_at ASC, created_at ASC
			FOR UPDATE SKIP LOCKED
			LIMIT $1
		),
		marked_events AS (
			UPDATE outbox_events o
			SET published_at = $2
			FROM locked
			WHERE o.id = locked.id
			RETURNING o.id, o.aggregate_type, o.aggregate_id, o.event_type,
				o.payload, o.available_at, o.created_at
		),
		marked_notifs AS (
			UPDATE notifications n
			SET status = $3, updated_at = $2
			FROM locked
			WHERE n.id = locked.aggregate_id AND n.status = $4
			RETURNING n.id
		)
		SELECT id, aggregate_type, aggregate_id, event_type, payload, available_at, created_at
		FROM marked_events
		ORDER BY available_at ASC, created_at ASC
	`, limit, now, model.StatusQueued, model.StatusPending)
	if err != nil {
		return nil, fmt.Errorf("claim outbox events: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	events := make([]OutboxEvent, 0, limit)
	for rows.Next() {
		var event OutboxEvent
		if err := rows.Scan(&event.ID, &event.AggregateType, &event.AggregateID, &event.EventType, &event.Payload, &event.AvailableAt, &event.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan outbox event: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate outbox events: %w", err)
	}
	return events, nil
}

func (r *notificationRepository) MarkOutboxEventPublished(ctx context.Context, eventID, notificationID string, publishedAt time.Time) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin publish tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	tag, err := tx.ExecContext(ctx, `UPDATE outbox_events SET published_at = $2 WHERE id = $1 AND published_at IS NULL`, eventID, publishedAt)
	if err != nil {
		return fmt.Errorf("mark outbox event published: %w", err)
	}
	rowsAffected, err := tag.RowsAffected()
	if err != nil {
		return fmt.Errorf("read outbox publish rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrOutboxEventNotFound
	}

	_, err = tx.ExecContext(ctx, `UPDATE notifications SET status = $2, updated_at = $3 WHERE id = $1 AND status = $4`, notificationID, model.StatusQueued, publishedAt, model.StatusPending)
	if err != nil {
		return fmt.Errorf("mark notification queued: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit publish tx: %w", err)
	}
	return nil
}

func (r *notificationRepository) StartProcessingNotification(ctx context.Context, id string, startedAt time.Time) (*model.Notification, error) {
	row := r.db.QueryRowContext(ctx, `
		UPDATE notifications
		SET status = $2, updated_at = $3
		WHERE id = $1 AND status = $4
		RETURNING id, batch_id, template_id, recipient, channel, content, priority, status, attempt_count, max_attempts, scheduled_at, next_attempt_at, provider_message_id, last_error, cancelled_at, created_at, updated_at
	`, id, model.StatusProcessing, startedAt, model.StatusQueued)
	notification, err := scanNotification(row.Scan)
	if err == nil {
		return &notification, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotificationNotReady
	}
	return nil, fmt.Errorf("start processing notification: %w", err)
}

func (r *notificationRepository) MarkNotificationSent(ctx context.Context, id string, attemptResult NotificationAttemptResult, providerMessageID string, sentAt time.Time) (*model.Notification, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin sent tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()
	row := tx.QueryRowContext(ctx, `
		UPDATE notifications
		SET status = $2, attempt_count = attempt_count + 1, provider_message_id = $3, last_error = NULL, next_attempt_at = NULL, updated_at = $4
		WHERE id = $1 AND status = $5
		RETURNING id, batch_id, template_id, recipient, channel, content, priority, status, attempt_count, max_attempts, scheduled_at, next_attempt_at, provider_message_id, last_error, cancelled_at, created_at, updated_at
	`, id, model.StatusSent, providerMessageID, sentAt, model.StatusProcessing)
	notification, err := scanNotification(row.Scan)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotificationNotReady
		}
		return nil, fmt.Errorf("mark notification sent: %w", err)
	}
	if err := r.insertNotificationAttempt(ctx, tx, notification.ID, notification.AttemptCount, attemptResult); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit sent tx: %w", err)
	}
	return &notification, nil
}

func (r *notificationRepository) MarkNotificationFailed(ctx context.Context, id string, attemptResult NotificationAttemptResult, failedAt time.Time) (*model.Notification, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin failed tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()
	row := tx.QueryRowContext(ctx, `
		UPDATE notifications
		SET status = $2, attempt_count = attempt_count + 1, last_error = $3, next_attempt_at = NULL, updated_at = $4
		WHERE id = $1 AND status = $5
		RETURNING id, batch_id, template_id, recipient, channel, content, priority, status, attempt_count, max_attempts, scheduled_at, next_attempt_at, provider_message_id, last_error, cancelled_at, created_at, updated_at
	`, id, model.StatusFailed, attemptResult.ErrorMessage, failedAt, model.StatusProcessing)
	notification, err := scanNotification(row.Scan)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotificationNotReady
		}
		return nil, fmt.Errorf("mark notification failed: %w", err)
	}
	if err := r.insertNotificationAttempt(ctx, tx, notification.ID, notification.AttemptCount, attemptResult); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit failed tx: %w", err)
	}
	return &notification, nil
}

func (r *notificationRepository) ScheduleNotificationRetry(ctx context.Context, id string, attemptResult NotificationAttemptResult, nextAttemptAt time.Time) (*model.Notification, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin retry tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()
	now := time.Now().UTC()
	row := tx.QueryRowContext(ctx, `
		UPDATE notifications
		SET status = $2, attempt_count = attempt_count + 1, last_error = $3, next_attempt_at = $4, updated_at = $5
		WHERE id = $1 AND status = $6
		RETURNING id, batch_id, template_id, recipient, channel, content, priority, status, attempt_count, max_attempts, scheduled_at, next_attempt_at, provider_message_id, last_error, cancelled_at, created_at, updated_at
	`, id, model.StatusPending, attemptResult.ErrorMessage, nextAttemptAt, now, model.StatusProcessing)
	notification, err := scanNotification(row.Scan)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotificationNotReady
		}
		return nil, fmt.Errorf("schedule notification retry: %w", err)
	}
	if err := r.insertNotificationAttempt(ctx, tx, notification.ID, notification.AttemptCount, attemptResult); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO outbox_events (id, aggregate_type, aggregate_id, event_type, payload, available_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		uuid.New(),
		"notification",
		notification.ID,
		"notification.retry_scheduled",
		[]byte(fmt.Sprintf(`{"notification_id":"%s","channel":"%s","priority":"%s"}`, notification.ID, notification.Channel, notification.Priority)),
		nextAttemptAt,
		now,
	); err != nil {
		return nil, fmt.Errorf("insert retry outbox event: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit retry tx: %w", err)
	}
	return &notification, nil
}

func (r *notificationRepository) insertIdempotencyKey(ctx context.Context, tx *sql.Tx, key, requestHash, batchID string) error {
	_, err := tx.ExecContext(
		ctx,
		`INSERT INTO idempotency_keys (id, idempotency_key, request_hash, batch_id, notification_id, created_at)
		 VALUES ($1, $2, $3, $4, NULL, $5)`,
		uuid.New(), key, requestHash, batchID, time.Now().UTC(),
	)
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrIdempotencyConflict
	}
	return fmt.Errorf("insert idempotency key: %w", err)
}

func (r *notificationRepository) insertNotificationIdempotencyKey(ctx context.Context, tx *sql.Tx, key, requestHash string, notificationID uuid.UUID) error {
	_, err := tx.ExecContext(
		ctx,
		`INSERT INTO idempotency_keys (id, idempotency_key, request_hash, batch_id, notification_id, created_at)
		 VALUES ($1, $2, $3, NULL, $4, $5)`,
		uuid.New(), key, requestHash, notificationID, time.Now().UTC(),
	)
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrIdempotencyConflict
	}
	return fmt.Errorf("insert notification idempotency key: %w", err)
}

func (r *notificationRepository) insertNotification(ctx context.Context, tx *sql.Tx, batchID *uuid.UUID, item request.NotificationItem, now time.Time) (model.Notification, error) {
	notificationID := uuid.New()
	priority := defaultPriority(item.Priority)
	var templateID any
	if item.TemplateID != nil {
		parsedTemplateID, err := uuid.Parse(*item.TemplateID)
		if err != nil {
			return model.Notification{}, fmt.Errorf("parse template id: %w", err)
		}
		templateID = parsedTemplateID
	}

	_, err := tx.ExecContext(
		ctx,
		`INSERT INTO notifications (
			id, batch_id, template_id, recipient, channel, content, priority, status,
			attempt_count, max_attempts, scheduled_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 0, 5, $9, $10, $10)`,
		notificationID, batchID, templateID, item.Recipient, item.Channel, item.Content, priority, model.StatusPending, item.ScheduledAt, now,
	)
	if err != nil {
		return model.Notification{}, fmt.Errorf("insert notification: %w", err)
	}

	payload := map[string]any{
		"notification_id": notificationID,
		"channel":         item.Channel,
		"priority":        priority,
	}
	if batchID != nil {
		payload["batch_id"] = batchID.String()
	}
	if traceCarrier := tracing.InjectCarrier(ctx); len(traceCarrier) > 0 {
		payload["trace_carrier"] = traceCarrier
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return model.Notification{}, fmt.Errorf("marshal outbox payload: %w", err)
	}

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO outbox_events (id, aggregate_type, aggregate_id, event_type, payload, available_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $6)`,
		uuid.New(), "notification", notificationID, "notification.created", payloadBytes, outboxAvailableAt(item.ScheduledAt, now),
	)
	if err != nil {
		return model.Notification{}, fmt.Errorf("insert outbox event: %w", err)
	}

	return model.Notification{
		ID:           notificationID,
		BatchID:      batchID,
		TemplateID:   parseOptionalUUID(item.TemplateID),
		Recipient:    item.Recipient,
		Channel:      item.Channel,
		Content:      item.Content,
		Priority:     priority,
		Status:       model.StatusPending,
		AttemptCount: 0,
		MaxAttempts:  5,
		ScheduledAt:  item.ScheduledAt,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

func (r *notificationRepository) loadExistingBatchIfPresent(ctx context.Context, tx *sql.Tx, key, requestHash string) (*model.NotificationBatch, []model.Notification, bool, bool, error) {
	var batchID sql.NullString
	var notificationID sql.NullString
	var storedHash string

	err := tx.QueryRowContext(ctx, `SELECT batch_id, notification_id, request_hash FROM idempotency_keys WHERE idempotency_key = $1`, key).
		Scan(&batchID, &notificationID, &storedHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, false, false, nil
		}
		return nil, nil, false, false, fmt.Errorf("load existing idempotency key: %w", err)
	}
	if storedHash != requestHash || !batchID.Valid {
		return nil, nil, false, true, ErrIdempotencyConflict
	}

	batch, notifications, replayed, err := r.loadBatchWithNotifications(ctx, tx, batchID.String, true)
	if err != nil {
		return nil, nil, false, true, err
	}
	return batch, notifications, replayed, true, nil
}

func (r *notificationRepository) loadExistingBatch(ctx context.Context, tx *sql.Tx, key, requestHash string) (*model.NotificationBatch, []model.Notification, bool, error) {
	batch, notifications, replayed, _, err := r.loadExistingBatchIfPresent(ctx, tx, key, requestHash)
	return batch, notifications, replayed, err
}

func (r *notificationRepository) loadExistingNotificationIfPresent(ctx context.Context, tx *sql.Tx, key, requestHash string) (model.Notification, bool, bool, error) {
	var batchID sql.NullString
	var notificationID sql.NullString
	var storedHash string

	err := tx.QueryRowContext(ctx, `SELECT batch_id, notification_id, request_hash FROM idempotency_keys WHERE idempotency_key = $1`, key).
		Scan(&batchID, &notificationID, &storedHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Notification{}, false, false, nil
		}
		return model.Notification{}, false, false, fmt.Errorf("load existing idempotency key: %w", err)
	}
	if storedHash != requestHash || !notificationID.Valid {
		return model.Notification{}, false, true, ErrIdempotencyConflict
	}

	notification, err := r.getNotificationByIDTx(ctx, tx, notificationID.String)
	if err != nil {
		return model.Notification{}, false, true, err
	}
	return notification, true, true, nil
}

func (r *notificationRepository) loadExistingNotification(ctx context.Context, tx *sql.Tx, key, requestHash string) (model.Notification, bool, error) {
	notification, replayed, _, err := r.loadExistingNotificationIfPresent(ctx, tx, key, requestHash)
	return notification, replayed, err
}

func (r *notificationRepository) loadBatchWithNotifications(ctx context.Context, tx *sql.Tx, batchID string, replayed bool) (*model.NotificationBatch, []model.Notification, bool, error) {
	var batch model.NotificationBatch
	err := tx.QueryRowContext(ctx, `SELECT id, idempotency_key, total_count, created_at FROM notification_batches WHERE id = $1`, batchID).
		Scan(&batch.ID, &batch.IdempotencyKey, &batch.TotalCount, &batch.CreatedAt)
	if err != nil {
		return nil, nil, false, fmt.Errorf("load notification batch: %w", err)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT id, batch_id, template_id, recipient, channel, content, priority, status, attempt_count, max_attempts, scheduled_at, next_attempt_at, provider_message_id, last_error, cancelled_at, created_at, updated_at
		FROM notifications
		WHERE batch_id = $1
		ORDER BY created_at ASC
	`, batchID)
	if err != nil {
		return nil, nil, false, fmt.Errorf("load notifications: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	notifications := make([]model.Notification, 0, batch.TotalCount)
	for rows.Next() {
		notification, err := scanNotification(rows.Scan)
		if err != nil {
			return nil, nil, false, fmt.Errorf("scan notification: %w", err)
		}
		notifications = append(notifications, notification)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, false, fmt.Errorf("iterate notifications: %w", err)
	}

	return &batch, notifications, replayed, nil
}

func defaultPriority(priority model.NotificationPriority) model.NotificationPriority {
	if priority == "" {
		return model.PriorityNormal
	}
	return priority
}

func nullableText(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func outboxAvailableAt(scheduledAt *time.Time, now time.Time) time.Time {
	if scheduledAt == nil {
		return now
	}
	return scheduledAt.UTC()
}

func (r *notificationRepository) getNotificationByIDTx(ctx context.Context, tx *sql.Tx, id string) (model.Notification, error) {
	notification, err := r.getNotificationByIDQuery(ctx, tx.QueryRowContext, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Notification{}, ErrNotificationNotFound
		}
		return model.Notification{}, fmt.Errorf("get notification by id: %w", err)
	}
	return notification, nil
}

type queryRowFunc func(ctx context.Context, query string, args ...any) *sql.Row

func (r *notificationRepository) getNotificationByIDQuery(ctx context.Context, queryRow queryRowFunc, id string) (model.Notification, error) {
	row := queryRow(ctx, `
		SELECT id, batch_id, template_id, recipient, channel, content, priority, status, attempt_count, max_attempts, scheduled_at, next_attempt_at, provider_message_id, last_error, cancelled_at, created_at, updated_at
		FROM notifications
		WHERE id = $1
	`, id)
	return scanNotification(row.Scan)
}

type scanner func(dest ...any) error

func scanNotification(scan scanner) (model.Notification, error) {
	var notification model.Notification
	var id string
	var batchID sql.NullString
	var templateID sql.NullString
	err := scan(
		&id, &batchID, &templateID, &notification.Recipient, &notification.Channel, &notification.Content, &notification.Priority, &notification.Status,
		&notification.AttemptCount, &notification.MaxAttempts, &notification.ScheduledAt, &notification.NextAttemptAt, &notification.ProviderMessageID,
		&notification.LastError, &notification.CancelledAt, &notification.CreatedAt, &notification.UpdatedAt,
	)
	if err != nil {
		return model.Notification{}, err
	}

	notification.ID, err = uuid.Parse(id)
	if err != nil {
		return model.Notification{}, fmt.Errorf("parse notification id: %w", err)
	}
	if batchID.Valid {
		parsedBatchID, parseErr := uuid.Parse(batchID.String)
		if parseErr != nil {
			return model.Notification{}, fmt.Errorf("parse batch id: %w", parseErr)
		}
		notification.BatchID = &parsedBatchID
	}
	if templateID.Valid {
		parsedTemplateID, parseErr := uuid.Parse(templateID.String)
		if parseErr != nil {
			return model.Notification{}, fmt.Errorf("parse template id: %w", parseErr)
		}
		notification.TemplateID = &parsedTemplateID
	}

	return notification, nil
}

func (r *notificationRepository) insertNotificationAttempt(ctx context.Context, tx *sql.Tx, notificationID uuid.UUID, attemptNumber int, result NotificationAttemptResult) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO notification_attempts (
			id, notification_id, attempt_number, response_status_code, response_body, error_message, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, uuid.New(), notificationID, attemptNumber, result.ResponseStatusCode, result.ResponseBody, result.ErrorMessage, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("insert notification attempt: %w", err)
	}
	return nil
}

func parseOptionalUUID(raw *string) *uuid.UUID {
	if raw == nil || *raw == "" {
		return nil
	}
	parsed, err := uuid.Parse(*raw)
	if err != nil {
		return nil
	}
	return &parsed
}
