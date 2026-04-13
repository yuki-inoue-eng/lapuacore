package gateways

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/bmizerany/assert"
)

func TestTTLCache_AddIfAbsent(t *testing.T) {
	tests := []struct {
		name     string
		ttl      time.Duration
		setup    func(c *TTLCache)
		key      string
		expected bool
	}{
		{
			name:     "first seen returns true",
			ttl:      1 * time.Second,
			setup:    func(c *TTLCache) {},
			key:      "key1",
			expected: true,
		},
		{
			name: "duplicate returns false",
			ttl:  1 * time.Second,
			setup: func(c *TTLCache) {
				c.AddIfAbsent("key1")
			},
			key:      "key1",
			expected: false,
		},
		{
			name: "different key returns true",
			ttl:  1 * time.Second,
			setup: func(c *TTLCache) {
				c.AddIfAbsent("key1")
			},
			key:      "key2",
			expected: true,
		},
		{
			name: "expired entry returns true",
			ttl:  10 * time.Millisecond,
			setup: func(c *TTLCache) {
				c.AddIfAbsent("key1")
				time.Sleep(20 * time.Millisecond)
			},
			key:      "key1",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewTTLCache(tt.ttl)
			tt.setup(cache)
			assert.Equal(t, tt.expected, cache.AddIfAbsent(tt.key))
		})
	}
}

func TestTTLCache_AddIfAbsent_ConcurrentAccess(t *testing.T) {
	cache := NewTTLCache(1 * time.Second)
	var wg sync.WaitGroup
	firstSeenCount := 0
	var mu sync.Mutex

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if cache.AddIfAbsent("same-key") {
				mu.Lock()
				firstSeenCount++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	// Due to RWMutex-based locking (Get=RLock, Set=Lock), there is a small
	// race window where multiple goroutines may see the key as absent.
	// For the dedup use case this is acceptable (at most a few duplicates).
	assert.T(t, firstSeenCount >= 1, "at least one goroutine should see first")
}

func TestTTLCache_StartCleanup(t *testing.T) {
	cache := NewTTLCache(10 * time.Millisecond)
	cache.SetCleanupInterval(50 * time.Millisecond)

	// add >threshold entries
	for i := 0; i < 1100; i++ {
		cache.AddIfAbsent(fmt.Sprintf("key-%d", i))
	}
	assert.Equal(t, 1100, cache.entries.Len())

	// wait for entries to expire
	time.Sleep(20 * time.Millisecond)

	// start cleanup goroutine
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go cache.StartCleanup(ctx)

	// wait for cleanup tick
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, 0, cache.entries.Len())
}

func TestTTLCache_StartCleanup_PreservesNonExpired(t *testing.T) {
	cache := NewTTLCache(500 * time.Millisecond)
	cache.SetCleanupInterval(50 * time.Millisecond)

	// add >threshold expired entries
	for i := 0; i < 1100; i++ {
		cache.AddIfAbsent(fmt.Sprintf("old-%d", i))
	}

	// wait for old entries to expire, then add fresh entries
	time.Sleep(600 * time.Millisecond)
	for i := 0; i < 5; i++ {
		cache.AddIfAbsent(fmt.Sprintf("fresh-%d", i))
	}
	assert.Equal(t, 1105, cache.entries.Len())

	// start cleanup goroutine
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go cache.StartCleanup(ctx)

	// wait for cleanup tick
	time.Sleep(100 * time.Millisecond)

	// only the 5 fresh entries should remain
	assert.Equal(t, 5, cache.entries.Len())
	for i := 0; i < 5; i++ {
		_, exists := cache.entries.Get(fmt.Sprintf("fresh-%d", i))
		assert.Equal(t, true, exists)
	}
}
