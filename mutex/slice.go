package mutex

import (
	"sort"
	"sync"
)

type Slice[T any] struct {
	mu     sync.RWMutex
	maxLen int // unlimited if 0 or less
	a      []T
}

func NewSlice[T any](src []T, maxLen int) *Slice[T] {
	dst := append([]T(nil), src...)
	if maxLen > 0 && len(dst) > maxLen {
		dst = dst[:maxLen]
	}
	return &Slice[T]{
		a:      dst,
		maxLen: maxLen,
	}
}

func (s *Slice[T]) AppendHead(item T) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.maxLen > 0 && len(s.a) >= s.maxLen {
		s.drop(len(s.a) - 1)
	}
	s.a = append([]T{item}, s.a...)
}

func (s *Slice[T]) Append(item T) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.maxLen > 0 && len(s.a) >= s.maxLen {
		s.drop(0)
	}
	s.a = append(s.a, item)
}

func (s *Slice[T]) Merge(s2 *Slice[T]) {
	s2.mu.RLock()
	tmp := append([]T(nil), s2.a...)
	s2.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	s.a = append(s.a, tmp...)
	if s.maxLen > 0 && len(s.a) > s.maxLen {
		s.a = append([]T(nil), s.a[len(s.a)-s.maxLen:]...)
	}
}

func (s *Slice[T]) Range(f func(int, T) bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i, item := range s.a {
		if !f(i, item) {
			break
		}
	}
}

func (s *Slice[T]) Drop(i int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if i < 0 || i >= len(s.a) {
		return false
	}
	s.drop(i)
	return true
}

func (s *Slice[T]) Drops(indices []int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.drops(indices)
}

func (s *Slice[T]) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.a)
}

func (s *Slice[T]) Get(i int) T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if i < 0 {
		i = len(s.a) + i
	}
	return s.a[i]
}

func (s *Slice[T]) drop(i int) {
	copy(s.a[i:], s.a[i+1:])
	var zero T
	s.a[len(s.a)-1] = zero
	s.a = s.a[:len(s.a)-1]
}

func (s *Slice[T]) drops(indices []int) {
	unique := uniqueSorted(indices)

	for removed, idx := range unique {
		adj := idx - removed
		if adj < 0 || adj >= len(s.a) {
			continue
		}
		copy(s.a[adj:], s.a[adj+1:])
		var zero T
		s.a[len(s.a)-1] = zero
		s.a = s.a[:len(s.a)-1]
	}
}

func uniqueSorted(indices []int) []int {
	cp := append([]int(nil), indices...)
	sort.Ints(cp)

	unique := make([]int, 0, len(cp))
	for i, idx := range cp {
		if i == 0 || cp[i-1] != idx {
			unique = append(unique, idx)
		}
	}
	return unique
}
