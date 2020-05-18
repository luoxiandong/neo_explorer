package util

import "sync/atomic"

type SafeCounter struct {
	val int32
}

func (s *SafeCounter) Get() int {
	return int(atomic.LoadInt32(&s.val))
}

func (s *SafeCounter) Set(val int) {
	atomic.StoreInt32(&s.val, int32(val))
}

func (s *SafeCounter) Add(delta int) int {
	return int(atomic.AddInt32(&s.val, int32(delta)))
}
