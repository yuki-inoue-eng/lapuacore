package agent

import (
	"fmt"
	"sync"
	"time"
)

// rateLimiter enforces Bybit short cycle rate limits.
// see: https://bybit-exchange.github.io/docs/v5/rate-limit

// RateLimitGroup represents a group of endpoints that share a rate limit quota.
type RateLimitGroup int

const (
	// GroupCreateOrder covers order creation endpoints (9r/1s).
	GroupCreateOrder RateLimitGroup = iota
	// GroupCancelOrder covers order cancellation endpoints (9r/1s).
	GroupCancelOrder
	// GroupAmendOrder covers order amendment endpoints (9r/1s).
	GroupAmendOrder
)

const (
	limitCreateOrder = 9
	limitCancelOrder = 9
	limitAmendOrder  = 9
)

type rateLimiter struct {
	mu       sync.Mutex
	quotas   map[RateLimitGroup]int
	capacity map[RateLimitGroup]int
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{
		quotas: map[RateLimitGroup]int{
			GroupCreateOrder: limitCreateOrder,
			GroupCancelOrder: limitCancelOrder,
			GroupAmendOrder:  limitAmendOrder,
		},
		capacity: map[RateLimitGroup]int{
			GroupCreateOrder: limitCreateOrder,
			GroupCancelOrder: limitCancelOrder,
			GroupAmendOrder:  limitAmendOrder,
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
