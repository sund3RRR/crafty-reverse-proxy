package proxy

import (
	"sync"
	"time"
)

type Timer struct {
	mu    *sync.Mutex
	timer *time.Timer
}

func NewTimer() *Timer {
	return &Timer{
		mu:    &sync.Mutex{},
		timer: nil,
	}
}

func (t *Timer) Schedule(timeout time.Duration, scheduleFunc func()) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.timer = time.AfterFunc(timeout, scheduleFunc)
}

func (t *Timer) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.timer != nil {
		t.timer.Stop()
	}
}
