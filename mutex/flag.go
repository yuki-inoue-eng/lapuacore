package mutex

import "sync"

type Flag struct {
	mu sync.RWMutex
	v  bool
}

func NewFlag(v bool) *Flag { return &Flag{v: v} }

func (f *Flag) Get() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.v
}

func (f *Flag) Set(v bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.v = v
}
