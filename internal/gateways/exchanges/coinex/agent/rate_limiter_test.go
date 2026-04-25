package agent

import (
	"testing"
	"time"
)

func TestRateLimiter_Consume(t *testing.T) {
	tests := []struct {
		name    string
		group   RateLimitGroup
		n       int
		wantErr bool
	}{
		{"order single", GroupOrder, 1, false},
		{"order at limit", GroupOrder, limitOrder, false},
		{"order over limit", GroupOrder, limitOrder + 1, true},
		{"cancel single", GroupCancel, 1, false},
		{"cancel at limit", GroupCancel, limitCancel, false},
		{"cancel over limit", GroupCancel, limitCancel + 1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := newRateLimiter()
			err := l.consume(tt.group, tt.n)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestRateLimiter_QuotaDecreases(t *testing.T) {
	l := newRateLimiter()

	if err := l.consume(GroupOrder, 5); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	l.mu.Lock()
	got := l.quotas[GroupOrder]
	l.mu.Unlock()
	if got != limitOrder-5 {
		t.Errorf("got %v, want %v", got, limitOrder-5)
	}
}

func TestRateLimiter_GroupIndependence(t *testing.T) {
	l := newRateLimiter()

	// Exhaust order quota
	if err := l.consume(GroupOrder, limitOrder); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Cancel quota should be unaffected
	if err := l.consume(GroupCancel, 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	l.mu.Lock()
	cancelQuota := l.quotas[GroupCancel]
	l.mu.Unlock()
	if got, want := cancelQuota, limitCancel-1; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestRateLimiter_QuotaRecovery(t *testing.T) {
	l := newRateLimiter()

	if err := l.consume(GroupOrder, limitOrder); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should fail immediately
	err := l.consume(GroupOrder, 1)
	if err == nil {
		t.Errorf("expected error but got nil")
	}

	// Wait for recovery
	time.Sleep(1100 * time.Millisecond)

	// Should succeed after recovery
	if err := l.consume(GroupOrder, 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	l.mu.Lock()
	got := l.quotas[GroupOrder]
	l.mu.Unlock()
	if got != limitOrder-1 {
		t.Errorf("got %v, want %v", got, limitOrder-1)
	}
}
