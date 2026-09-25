package api

import (
	"sync"
	"time"
)

// limiter allows n events per key within a fixed window. Keys are dropped
// when their window ends, so memory stays bounded by recent clients.
type limiter struct {
	mu     sync.Mutex
	n      int
	window time.Duration
	now    func() time.Time
	seen   map[string]*bucket
}

type bucket struct {
	start time.Time
	count int
}

func newLimiter(n int, window time.Duration, now func() time.Time) *limiter {
	return &limiter{n: n, window: window, now: now, seen: map[string]*bucket{}}
}

func (l *limiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	t := l.now()
	for k, b := range l.seen {
		if t.Sub(b.start) >= l.window {
			delete(l.seen, k)
		}
	}
	b := l.seen[key]
	if b == nil {
		b = &bucket{start: t}
		l.seen[key] = b
	}
	if b.count >= l.n {
		return false
	}
	b.count++
	return true
}
