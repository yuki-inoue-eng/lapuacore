package gateways

import (
	"context"
	"time"

	"github.com/yuki-inoue-eng/lapuacore/mutex"
)

const (
	cleanupThreshold  = 1000
	cleanupIntervalSc = 1 // seconds
)

// TTLCache is a thread-safe cache that automatically expires entries after a TTL.
// Uses mutex.Map (RWMutex-based) so reads (AddIfAbsent checks) can proceed
// concurrently while cleanup only takes a write lock for the final BulkDelete.
type TTLCache struct {
	entries         *mutex.Map[string, time.Time]
	ttl             time.Duration
	cleanupInterval time.Duration
}

// NewTTLCache creates a new TTL cache with the given expiry duration.
func NewTTLCache(ttl time.Duration) *TTLCache {
	return &TTLCache{
		entries:         mutex.NewMap[string, time.Time](nil),
		ttl:             ttl,
		cleanupInterval: cleanupIntervalSc * time.Second,
	}
}

// SetCleanupInterval overrides the default cleanup interval (1s).
func (c *TTLCache) SetCleanupInterval(d time.Duration) {
	c.cleanupInterval = d
}

// AddIfAbsent returns true if the key was newly added (first seen).
// Returns false if the key already exists and has not expired (duplicate).
// Uses SetIfAbsent to atomically check and insert under a single lock.
func (c *TTLCache) AddIfAbsent(key string) bool {
	now := time.Now()
	existing, occupied := c.entries.SetIfAbsent(key, now)
	if !occupied {
		return true // newly inserted
	}
	if now.Sub(existing) > c.ttl {
		// expired — overwrite with fresh timestamp
		c.entries.Set(key, now)
		return true
	}
	return false // duplicate
}

// StartCleanup runs a background goroutine that periodically removes expired
// entries when the cache exceeds the cleanup threshold. Cancel ctx to stop.
func (c *TTLCache) StartCleanup(ctx context.Context) {
	ticker := time.NewTicker(c.cleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if c.entries.Len() <= cleanupThreshold {
				continue
			}
			// Snapshot keys with a brief RLock, then check each
			// individually so the read lock is not held during the scan.
			keys := c.entries.GetKeys()
			now := time.Now()
			var expiredKeys []string
			for _, key := range keys {
				if t, ok := c.entries.Get(key); ok && now.Sub(t) > c.ttl {
					expiredKeys = append(expiredKeys, key)
				}
			}
			if len(expiredKeys) > 0 {
				c.entries.BulkDelete(expiredKeys)
			}
		}
	}
}
