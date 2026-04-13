package agent

import (
	"fmt"
	"sync"
	"time"
)

// rateLimiter enforces Bybit API rate limits (rolling window per second per UID).
// see: https://bybit-exchange.github.io/docs/v5/rate-limit

// RateLimitGroup represents a group of endpoints that share a rate limit quota.
type RateLimitGroup int

const (
	// GroupCreateOrder covers order creation endpoints (20r/1s for Linear).
	GroupCreateOrder RateLimitGroup = iota
	// GroupCancelOrder covers order cancellation endpoints (20r/1s for Linear).
	GroupCancelOrder
	// GroupAmendOrder covers order amendment endpoints (10r/1s for Linear).
	GroupAmendOrder
)

// Linear rate limits per the Bybit V5 API documentation.
const (
	limitCreateOrder = 20
	limitCancelOrder = 20
	limitAmendOrder  = 10
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
