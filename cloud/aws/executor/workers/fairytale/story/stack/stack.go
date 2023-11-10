package stack

import (
	"sync"
	"sync/atomic"

	log "github.com/cantara/bragi/sbragi"
)

type Stack[T any] interface {
	Pop() (T, bool)
	Push(T)
}

type stack[T any] struct {
	data   []T
	len    atomic.Int64
	expand func()
}

func New[T any]() Stack[T] {
	s := &stack[T]{}
	s.expand = expand(s, 5)
	return s
}

func (s *stack[T]) Pop() (v T, ok bool) {
	l := s.len.Add(-1)
	if l <= 0 {
		s.len.CompareAndSwap(l, 0)
		ok = false
		return
	}
	v = s.data[l-1]
	return
}

func (s *stack[T]) Push(v T) {
	l := s.len.Add(1)
	log.Info("pushing", "v", v, "len", len(s.data), "cap", cap(s.data))
	if cap(s.data) < int(l) {
		s.expand()
	}
	log.Info("pushing", "v", v, "len", len(s.data), "cap", cap(s.data))
	s.data[l-1] = v
	//s.data = append(s.data, v)
}

func expand[T any](s *stack[T], n int) func() {
	return sync.OnceFunc(func() {
		log.Info("expanding")
		s.data = append(s.data, make([]T, n)...) //append(make([]T, 0, len(s.data)+n), s.data...)
		s.expand = expand(s, n)
	})
}
