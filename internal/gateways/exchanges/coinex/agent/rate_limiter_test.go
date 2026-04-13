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
				assert.NotEqual(t, nil, err)
			} else {
				assert.Equal(t, nil, err)
			}
		})
	}
}

func TestRateLimiter_QuotaDecreases(t *testing.T) {
	l := newRateLimiter()

	assert.Equal(t, nil, l.consume(GroupOrder, 5))
	assert.Equal(t, limitOrder-5, l.quotas[GroupOrder])
}

func TestRateLimiter_GroupIndependence(t *testing.T) {
	l := newRateLimiter()

	// Exhaust order quota
	assert.Equal(t, nil, l.consume(GroupOrder, limitOrder))

	// Cancel quota should be unaffected
	assert.Equal(t, nil, l.consume(GroupCancel, 1))
	assert.Equal(t, limitCancel-1, l.quotas[GroupCancel])
}

func TestRateLimiter_QuotaRecovery(t *testing.T) {
	l := newRateLimiter()

	assert.Equal(t, nil, l.consume(GroupOrder, limitOrder))

	// Should fail immediately
	err := l.consume(GroupOrder, 1)
	assert.NotEqual(t, nil, err)

	// Wait for recovery
	time.Sleep(1100 * time.Millisecond)

	// Should succeed after recovery
	assert.Equal(t, nil, l.consume(GroupOrder, 1))
	assert.Equal(t, limitOrder-1, l.quotas[GroupOrder])
}
