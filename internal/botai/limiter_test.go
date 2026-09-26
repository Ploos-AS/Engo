package botai

import (
	"testing"
	"time"
)

func TestLimiterConcurrency(t *testing.T) {
	l := NewLimiter(1, 0)
	release, ok := l.TryAcquire("a")
	if !ok {
		t.Fatal("first acquire failed")
	}
	if _, ok := l.TryAcquire("b"); ok {
		t.Fatal("concurrency limit not enforced")
	}
	release()
	if release2, ok := l.TryAcquire("b"); !ok {
		t.Fatal("slot not released")
	} else {
		release2()
	}
}
func TestLimiterCooldownPerScope(t *testing.T) {
	l := NewLimiter(2, time.Hour)
	r, ok := l.TryAcquire("a")
	if !ok {
		t.Fatal("first acquire failed")
	}
	r()
	if _, ok := l.TryAcquire("a"); ok {
		t.Fatal("cooldown not enforced")
	}
	if r, ok := l.TryAcquire("b"); !ok {
		t.Fatal("cooldown leaked across scopes")
	} else {
		r()
	}
}
