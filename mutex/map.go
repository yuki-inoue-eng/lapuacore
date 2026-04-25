package mutex

import (
	"sync"
)

type Map[K comparable, V any] struct {
	mu sync.RWMutex
	mm map[K]V
}

func NewMap[K comparable, V any](m map[K]V) *Map[K, V] {
	if m == nil {
		return &Map[K, V]{mm: map[K]V{}}
	} else {
		return &Map[K, V]{mm: m}
	}
}

func (m *Map[K, V]) GetKeys() []K {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var r []K
	for k := range m.mm {
		r = append(r, k)
	}
	return r
}

func (m *Map[K, V]) Values() []V {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var r []V
	for _, v := range m.mm {
		r = append(r, v)
	}
	return r
}

func (m *Map[K, V]) Set(key K, value V) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mm[key] = value
}

// SetIfAbsent sets the value only if the key does not already exist.
// Returns the existing value and true if the key was present, or the zero value and false if newly inserted.
func (m *Map[K, V]) SetIfAbsent(key K, value V) (V, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if existing, ok := m.mm[key]; ok {
		return existing, true
	}
	m.mm[key] = value
	var zero V
	return zero, false
}

func (m *Map[K, V]) Get(key K) (V, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	value, ok := m.mm[key]
	return value, ok
}

// Gets returns a list of the elements specified in the arguments.
func (m *Map[K, V]) Gets(keys []K) []V {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []V
	for i := range keys {
		result = append(result, m.mm[keys[i]])
	}
	return result
}

func (m *Map[K, V]) Delete(key K) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.mm, key)
}

func (m *Map[K, V]) BulkDelete(keys []K) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, k := range keys {
		delete(m.mm, k)
	}
}

func (m *Map[K, V]) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.mm)
}

func (m *Map[K, V]) Range(f func(K, V) bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for k, v := range m.mm {
		if !f(k, v) {
			break
		}
	}
}
