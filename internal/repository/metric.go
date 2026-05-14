package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	postgresdb "insider-one-assessment/pkgs/postgres"
)

type MetricSnapshot struct {
	GeneratedAt      time.Time      `json:"generated_at"`
	QueueDepth       int            `json:"queue_depth"`
	SuccessCount     int            `json:"success_count"`
	FailureCount     int            `json:"failure_count"`
	CancelledCount   int            `json:"cancelled_count"`
	PendingCount     int            `json:"pending_count"`
	QueuedCount      int            `json:"queued_count"`
	ProcessingCount  int            `json:"processing_count"`
	AverageLatencyMs float64        `json:"average_latency_ms"`
	CountsByChannel  map[string]int `json:"counts_by_channel"`
}

type IMetricRepository interface {
	LoadSnapshot(ctx context.Context) (*MetricSnapshot, error)
}

type metricRepository struct {
	db *sql.DB
}

func NewMetricRepository(client postgresdb.IPostgresInstance) IMetricRepository {
	return &metricRepository{db: client.SQLDatabase()}
}

func (r *metricRepository) LoadSnapshot(ctx context.Context) (*MetricSnapshot, error) {
	snapshot := MetricSnapshot{
		GeneratedAt:     time.Now().UTC(),
		CountsByChannel: map[string]int{},
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT status, COUNT(*)
		FROM notifications
		GROUP BY status
	`)
	if err != nil {
		return nil, fmt.Errorf("load status counts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("scan status count: %w", err)
		}
		switch status {
		case "pending":
			snapshot.PendingCount = count
		case "queued":
			snapshot.QueuedCount = count
		case "processing":
			snapshot.ProcessingCount = count
		case "sent":
			snapshot.SuccessCount = count
		case "failed":
			snapshot.FailureCount = count
		case "cancelled":
			snapshot.CancelledCount = count
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate status counts: %w", err)
	}

	snapshot.QueueDepth = snapshot.PendingCount + snapshot.QueuedCount + snapshot.ProcessingCount

	channelRows, err := r.db.QueryContext(ctx, `
		SELECT channel, COUNT(*)
		FROM notifications
		GROUP BY channel
	`)
	if err != nil {
		return nil, fmt.Errorf("load channel counts: %w", err)
	}
	defer channelRows.Close()

	for channelRows.Next() {
		var channel string
		var count int
		if err := channelRows.Scan(&channel, &count); err != nil {
			return nil, fmt.Errorf("scan channel count: %w", err)
		}
		snapshot.CountsByChannel[channel] = count
	}

	if err := channelRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate channel counts: %w", err)
	}

	if err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (updated_at - created_at)) * 1000), 0)
		FROM notifications
		WHERE status IN ('sent', 'failed')
	`).Scan(&snapshot.AverageLatencyMs); err != nil {
		return nil, fmt.Errorf("load average latency: %w", err)
	}

	return &snapshot, nil
}
