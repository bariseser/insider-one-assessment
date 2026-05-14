package worker

import (
	"math/rand"
	"testing"
	"time"
)

func TestClassifyProviderStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		expected   retryDecision
	}{
		{name: "accepted", statusCode: 202, expected: retryDecisionNonRetryable},
		{name: "too many requests", statusCode: 429, expected: retryDecisionRetryableProviderError},
		{name: "server error", statusCode: 503, expected: retryDecisionRetryableProviderError},
		{name: "bad request", statusCode: 400, expected: retryDecisionNonRetryable},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := classifyProviderStatus(tt.statusCode)
			if actual != tt.expected {
				t.Fatalf("expected %v, got %v", tt.expected, actual)
			}
		})
	}
}

func TestComputeRetryDelay(t *testing.T) {
	t.Parallel()

	w := &Worker{
		rng: rand.New(rand.NewSource(1)),
	}

	delay1 := w.computeRetryDelay(1)
	delay2 := w.computeRetryDelay(2)
	delay5 := w.computeRetryDelay(5)
	delay10 := w.computeRetryDelay(10)

	if delay1 < time.Second || delay1 > 10*time.Second {
		t.Fatalf("unexpected attempt 1 delay: %s", delay1)
	}

	if delay2 <= delay1 {
		t.Fatalf("expected attempt 2 delay to be greater than attempt 1, got %s <= %s", delay2, delay1)
	}

	if delay5 <= delay2 {
		t.Fatalf("expected attempt 5 delay to be greater than attempt 2, got %s <= %s", delay5, delay2)
	}

	if delay10 > 5*time.Minute {
		t.Fatalf("expected capped delay <= 5m, got %s", delay10)
	}
}
