package agent

import (
	"testing"
	"time"

	"github.com/bmizerany/assert"
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
				assert.NotEqual(t, nil, err)
			} else {
				assert.Equal(t, nil, err)
			}
		})
	}
}

func TestRateLimiter_QuotaDecreases(t *testing.T) {
	l := newRateLimiter()

	assert.Equal(t, nil, l.consume(GroupCreateOrder, 5))
	assert.Equal(t, limitCreateOrder-5, l.quotas[GroupCreateOrder])
}

func TestRateLimiter_GroupIndependence(t *testing.T) {
	l := newRateLimiter()

	// Exhaust create quota
	assert.Equal(t, nil, l.consume(GroupCreateOrder, limitCreateOrder))

	// Cancel and amend quotas should be unaffected
	assert.Equal(t, nil, l.consume(GroupCancelOrder, 1))
	assert.Equal(t, limitCancelOrder-1, l.quotas[GroupCancelOrder])

	assert.Equal(t, nil, l.consume(GroupAmendOrder, 1))
	assert.Equal(t, limitAmendOrder-1, l.quotas[GroupAmendOrder])
}

func TestRateLimiter_QuotaRecovery(t *testing.T) {
	l := newRateLimiter()

	assert.Equal(t, nil, l.consume(GroupCreateOrder, limitCreateOrder))

	// Should fail immediately
	err := l.consume(GroupCreateOrder, 1)
	assert.NotEqual(t, nil, err)

	// Wait for recovery
	time.Sleep(1100 * time.Millisecond)

	// Should succeed after recovery
	assert.Equal(t, nil, l.consume(GroupCreateOrder, 1))
	assert.Equal(t, limitCreateOrder-1, l.quotas[GroupCreateOrder])
}
