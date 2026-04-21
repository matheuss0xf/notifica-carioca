package runtime

import (
	"context"
	"errors"
	"testing"
)

func TestReadinessCheckerCheck(t *testing.T) {
	tests := []struct {
		name      string
		pingDB    func(context.Context) error
		pingRedis func(context.Context) error
		wantReady bool
		wantWS    int
		wantErr   bool
	}{
		{
			name:      "all dependencies ready",
			pingDB:    func(context.Context) error { return nil },
			pingRedis: func(context.Context) error { return nil },
			wantReady: true,
			wantWS:    3,
		},
		{
			name:      "postgres failure",
			pingDB:    func(context.Context) error { return errors.New("pg down") },
			pingRedis: func(context.Context) error { return nil },
			wantWS:    3,
			wantErr:   true,
		},
		{
			name:      "redis failure",
			pingDB:    func(context.Context) error { return nil },
			pingRedis: func(context.Context) error { return errors.New("redis down") },
			wantWS:    3,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := &ReadinessChecker{
				pingDB:        tt.pingDB,
				pingRedis:     tt.pingRedis,
				wsConnections: func() int { return 3 },
			}

			ready, wsConnections, err := checker.Check(context.Background())
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ready != tt.wantReady {
				t.Fatalf("expected ready=%v, got %v", tt.wantReady, ready)
			}
			if wsConnections != tt.wantWS {
				t.Fatalf("expected ws connections %d, got %d", tt.wantWS, wsConnections)
			}
		})
	}
}
