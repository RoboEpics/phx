package util

import (
	"crypto/rand"
	"sync"
)

func All(args ...bool) bool {
	for _, a := range args {
		if !a {
			return false
		}
	}
	return true
}

func Any(args ...bool) bool {
	for _, a := range args {
		if a {
			return true
		}
	}
	return false
}

func EqualMap[K, V comparable](a, b map[K]V) bool {
	return EqualMapFunc(
		a, b,
		func(v1, v2 V) bool {
			return v1 == v2
		})
}

func EqualMapFunc[K comparable, V any](a, b map[K]V, eq func(v1, v2 V) bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		v2, ok := b[k]
		if !ok {
			return false
		}
		if !eq(v, v2) {
			return false
		}
	}
	return true
}

func EqualSlice[T comparable](a, b []T) bool {
	return EqualSliceFunc(
		a, b,
		func(v1, v2 T) bool {
			return v1 == v2
		})
}

func EqualSliceFunc[T any](a, b []T, eq func(v1, v2 T) bool) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !eq(a[i], b[i]) {
			return false
		}
	}
	return true
}

func EqualPointer[T comparable](a, b *T) bool {
	if a == nil && b == nil {
		return true
	}
	if a != nil || b != nil {
		return false
	}
	return *a == *b
}

// Is a <= b?
func SubsetMap[K, V comparable](a, b map[K]V) bool {
	for k, v := range a {
		v2, ok := b[k]
		if !ok {
			return false
		}
		if v != v2 {
			return false
		}
	}
	return true
}

func ReplaceValueMap[K, V comparable](m map[K]V, replace map[V]V) map[K]V {
	out := make(map[K]V)
	for k, v := range m {
		rv, ok := replace[v]
		if !ok {
			rv = v
		}
		out[k] = rv
	}
	return out
}

func Combinate[T any](args ...[]T) [][]T {
	if len(args) <= 0 {
		return nil
	}
	if len(args) <= 1 {
		out := make([][]T, len(args[0]))
		for i, x := range args[0] {
			out[i] = []T{x}
		}
		return out
	}
	out := make([][]T, 0)
	child := Combinate(args[1:]...)
	for _, new := range args[0] {
		lines := [][]T{}
		for _, items := range child {
			line := append([]T{new}, items...)
			lines = append(lines, line)
		}
		out = append(out, lines...)
	}
	return out
}

type Set[T comparable] map[T]struct{}

func (s *Set[T]) Put(x T) {
	(*s)[x] = struct{}{}
}

func (s *Set[T]) Has(x T) bool {
	if s == nil {
		return false
	}
	_, ok := (*s)[x]
	return ok
}

func (s *Set[T]) Range(fn func(x T) (result any)) any {
	if s == nil {
		return nil
	}
	for k := range *s {
		if result := fn(k); result != nil {
			return result
		}
	}
	return nil
}

var (
	CharsetHex = "ABCDEF1234567890"
	CharsetNum = "1234567890"
)

func RandomStr(charset string, size int) string {
	var (
		idxs = make([]byte, size)
		out  = make([]byte, size)
	)
	_, err := rand.Read(idxs)
	if err != nil {
		panic(err)
	}
	for i := range out {
		out[i] = charset[int(idxs[i])%len(charset)]
	}
	return string(out)
}

type Pubsub[T any] struct {
	chs     map[uint]chan T
	counter uint
	mu      sync.RWMutex
}

func (ps *Pubsub[T]) Register() (uint, <-chan T) {
	return ps.RegisterN(256)
}

func (ps *Pubsub[T]) RegisterN(n int) (uint, <-chan T) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if ps.chs == nil {
		ps.chs = make(map[uint]chan T)
	}
	ps.counter++
	ch := make(chan T, n)
	ps.chs[ps.counter] = ch
	return ps.counter, ch
}

func (ps *Pubsub[T]) Close(id uint) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	close(ps.chs[id])
	delete(ps.chs, id)
}

func (ps *Pubsub[T]) Broadcast(msg T) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	for _, ch := range ps.chs {
		ch <- msg
	}
}
