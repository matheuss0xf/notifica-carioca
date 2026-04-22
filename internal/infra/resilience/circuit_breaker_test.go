package resilience

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCircuitBreakerTripsAndRecovers(t *testing.T) {
	now := time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC)
	breaker := NewCircuitBreaker(2, 10*time.Second)
	breaker.now = func() time.Time { return now }

	downErr := errors.New("down")

	if err := breaker.Execute(context.Background(), func(context.Context) error { return downErr }); !errors.Is(err, downErr) {
		t.Fatalf("expected downstream error, got %v", err)
	}

	if err := breaker.Execute(context.Background(), func(context.Context) error { return downErr }); !errors.Is(err, downErr) {
		t.Fatalf("expected downstream error, got %v", err)
	}

	if err := breaker.Execute(context.Background(), func(context.Context) error { return nil }); !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("expected circuit open error, got %v", err)
	}

	now = now.Add(11 * time.Second)
	if err := breaker.Execute(context.Background(), func(context.Context) error { return nil }); err != nil {
		t.Fatalf("expected half-open request to succeed, got %v", err)
	}

	if err := breaker.Execute(context.Background(), func(context.Context) error { return nil }); err != nil {
		t.Fatalf("expected closed breaker after recovery, got %v", err)
	}
}
