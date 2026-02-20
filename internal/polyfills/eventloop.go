// Adapted from github.com/cryguy/hostedat/internal/worker/eventloop.go
// Copyright (c) cryguy/hostedat contributors. MIT License.
// See THIRD_PARTY_LICENSES for full license text.
package polyfills

import (
	"sync"
	"time"

	v8 "github.com/tommie/v8go"
)

// timerEntry represents a pending setTimeout or setInterval callback.
type timerEntry struct {
	callback *v8.Function
	deadline time.Time
	interval time.Duration // 0 for setTimeout, >0 for setInterval
	id       int
	cleared  bool
}

// EventLoop manages Go-backed timers for setTimeout/setInterval.
type EventLoop struct {
	mu     sync.Mutex
	timers map[int]*timerEntry
	nextID int
}

func NewEventLoop() *EventLoop {
	return &EventLoop{
		timers: make(map[int]*timerEntry),
	}
}

func (el *EventLoop) setTimeout(callback *v8.Function, delay time.Duration) int {
	el.mu.Lock()
	defer el.mu.Unlock()
	el.nextID++
	id := el.nextID
	el.timers[id] = &timerEntry{
		callback: callback,
		deadline: time.Now().Add(delay),
		id:       id,
	}
	return id
}

func (el *EventLoop) setInterval(callback *v8.Function, interval time.Duration) int {
	el.mu.Lock()
	defer el.mu.Unlock()
	el.nextID++
	id := el.nextID
	el.timers[id] = &timerEntry{
		callback: callback,
		deadline: time.Now().Add(interval),
		interval: interval,
		id:       id,
	}
	return id
}

func (el *EventLoop) clearTimer(id int) {
	el.mu.Lock()
	defer el.mu.Unlock()
	if t, ok := el.timers[id]; ok {
		t.cleared = true
		delete(el.timers, id)
	}
}

// Drain fires all pending timers until none remain or the deadline is exceeded.
// Must be called on the same goroutine as V8.
func (el *EventLoop) Drain(iso *v8.Isolate, ctx *v8.Context, deadline time.Time) {
	for {
		el.mu.Lock()
		if len(el.timers) == 0 {
			el.mu.Unlock()
			return
		}
		var next *timerEntry
		for _, t := range el.timers {
			if t.cleared {
				continue
			}
			if next == nil || t.deadline.Before(next.deadline) {
				next = t
			}
		}
		el.mu.Unlock()

		if next == nil {
			return
		}

		now := time.Now()
		if next.deadline.After(now) {
			wait := next.deadline.Sub(now)
			if now.Add(wait).After(deadline) {
				return
			}
			time.Sleep(wait)
		}

		if time.Now().After(deadline) {
			return
		}

		el.mu.Lock()
		if next.cleared {
			el.mu.Unlock()
			continue
		}
		if next.interval > 0 {
			next.deadline = time.Now().Add(next.interval)
		} else {
			delete(el.timers, next.id)
		}
		cb := next.callback
		el.mu.Unlock()

		undefinedVal := v8.Undefined(iso)
		_, _ = cb.Call(undefinedVal, undefinedVal)
		ctx.PerformMicrotaskCheckpoint()
	}
}
