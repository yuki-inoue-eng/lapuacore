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
		{"create single", GroupCreateOrder, 1, false},
		{"create at limit", GroupCreateOrder, limitCreateOrder, false},
		{"create over limit", GroupCreateOrder, limitCreateOrder + 1, true},
		{"cancel single", GroupCancelOrder, 1, false},
		{"cancel at limit", GroupCancelOrder, limitCancelOrder, false},
		{"cancel over limit", GroupCancelOrder, limitCancelOrder + 1, true},
		{"amend single", GroupAmendOrder, 1, false},
		{"amend at limit", GroupAmendOrder, limitAmendOrder, false},
		{"amend over limit", GroupAmendOrder, limitAmendOrder + 1, true},
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

	if err := l.consume(GroupCreateOrder, 5); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	l.mu.Lock()
	got := l.quotas[GroupCreateOrder]
	l.mu.Unlock()
	if got != limitCreateOrder-5 {
		t.Errorf("got %v, want %v", got, limitCreateOrder-5)
	}
}

func TestRateLimiter_GroupIndependence(t *testing.T) {
	l := newRateLimiter()

	// Exhaust create quota
	if err := l.consume(GroupCreateOrder, limitCreateOrder); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Cancel and amend quotas should be unaffected
	if err := l.consume(GroupCancelOrder, 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	l.mu.Lock()
	cancelQuota := l.quotas[GroupCancelOrder]
	l.mu.Unlock()
	if got, want := cancelQuota, limitCancelOrder-1; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if err := l.consume(GroupAmendOrder, 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	l.mu.Lock()
	amendQuota := l.quotas[GroupAmendOrder]
	l.mu.Unlock()
	if got, want := amendQuota, limitAmendOrder-1; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestRateLimiter_QuotaRecovery(t *testing.T) {
	l := newRateLimiter()

	if err := l.consume(GroupCreateOrder, limitCreateOrder); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should fail immediately
	err := l.consume(GroupCreateOrder, 1)
	if err == nil {
		t.Errorf("expected error but got nil")
	}

	// Wait for recovery
	time.Sleep(1100 * time.Millisecond)

	// Should succeed after recovery
	if err := l.consume(GroupCreateOrder, 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	l.mu.Lock()
	got := l.quotas[GroupCreateOrder]
	l.mu.Unlock()
	if got != limitCreateOrder-1 {
		t.Errorf("got %v, want %v", got, limitCreateOrder-1)
	}
}
