package agent

import (
	"fmt"
	"sync"
	"time"
)

// rateLimiter enforces CoinEx short cycle rate limits.
// see: https://docs.coinex.com/api/v2/rate-limit

// RateLimitGroup represents a group of endpoints that share a rate limit quota.
type RateLimitGroup int

const (
	// GroupOrder covers place, batch-place, and modify order endpoints (20r/1s).
	GroupOrder RateLimitGroup = iota
	// GroupCancel covers cancel and batch-cancel order endpoints (40r/1s).
	GroupCancel
)

const (
	limitOrder  = 20
	limitCancel = 40
)

type rateLimiter struct {
	mu       sync.Mutex
	quotas   map[RateLimitGroup]int
	capacity map[RateLimitGroup]int
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{
		quotas: map[RateLimitGroup]int{
			GroupOrder:  limitOrder,
			GroupCancel: limitCancel,
		},
		capacity: map[RateLimitGroup]int{
			GroupOrder:  limitOrder,
			GroupCancel: limitCancel,
		},
	}
}

// consume attempts to consume n quota units from the given group.
// Returns an error if insufficient quota is available.
func (l *rateLimiter) consume(group RateLimitGroup, n int) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.quotas[group]-n < 0 {
		return fmt.Errorf("rate limit reached (group=%d, remaining=%d, requested=%d)", group, l.quotas[group], n)
	}
	l.quotas[group] -= n

	go func() {
		time.Sleep(1 * time.Second)
		l.mu.Lock()
		defer l.mu.Unlock()
		l.quotas[group] += n
		if l.quotas[group] > l.capacity[group] {
			l.quotas[group] = l.capacity[group]
		}
	}()

	return nil
}
