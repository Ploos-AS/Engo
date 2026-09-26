package botai

import (
	"sync"
	"time"
)

type Limiter struct {
	mu       sync.Mutex
	slots    chan struct{}
	cooldown time.Duration
	last     map[string]time.Time
}

func NewLimiter(maxConcurrent int, cooldown time.Duration) *Limiter {
	if maxConcurrent < 1 {
		maxConcurrent = 1
	}
	if cooldown < 0 {
		cooldown = 0
	}
	return &Limiter{slots: make(chan struct{}, maxConcurrent), cooldown: cooldown, last: make(map[string]time.Time)}
}

func (l *Limiter) TryAcquire(key string) (func(), bool) {
	now := time.Now()
	l.mu.Lock()
	if last := l.last[key]; !last.IsZero() && now.Sub(last) < l.cooldown {
		l.mu.Unlock()
		return nil, false
	}
	select {
	case l.slots <- struct{}{}:
		l.last[key] = now
		l.mu.Unlock()
		var once sync.Once
		return func() { once.Do(func() { <-l.slots }) }, true
	default:
		l.mu.Unlock()
		return nil, false
	}
}
