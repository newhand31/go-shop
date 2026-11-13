// Package ratelimit is a small in-memory token-bucket limiter, one bucket per key (for example a client IP).
//
// It is per process: with two API instances each one keeps its own buckets. Sharing the counters through Redis is
// possible later; for one instance, a map and a mutex are enough.
package ratelimit

import (
	"math"
	"sync"
	"time"
)

// Limiter hands out tokens at a steady rate and lets a key save up at most `burst` of them.
type Limiter struct {
	rate  float64 // tokens added per second
	burst float64 // bucket capacity
	now   func() time.Time

	mu        sync.Mutex
	buckets   map[string]*bucket
	lastSweep time.Time
}

type bucket struct {
	tokens float64
	last   time.Time
}

// New returns a Limiter that allows `rate` requests per second per key, with bursts up to `burst`.
func New(rate float64, burst int) *Limiter {
	return newWithClock(rate, burst, time.Now)
}

// newWithClock lets tests drive time by hand instead of sleeping.
func newWithClock(rate float64, burst int, now func() time.Time) *Limiter {
	return &Limiter{rate: rate, burst: float64(burst), now: now, buckets: make(map[string]*bucket), lastSweep: now()}
}

// Allow takes one token for key. When the bucket is empty it returns false and how long until a token is available.
func (l *Limiter) Allow(key string) (ok bool, retryAfter time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.sweep(now)

	b, found := l.buckets[key]
	if !found {
		b = &bucket{tokens: l.burst, last: now}
		l.buckets[key] = b
	}
	b.tokens = math.Min(l.burst, b.tokens+now.Sub(b.last).Seconds()*l.rate)
	b.last = now

	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}
	return false, time.Duration((1 - b.tokens) / l.rate * float64(time.Second))
}

// sweep drops buckets that have been idle long enough to be full again; a full bucket is the same as no bucket.
// Without it, one request from each of a million IPs would stay in the map forever. Caller holds l.mu.
func (l *Limiter) sweep(now time.Time) {
	idle := time.Duration(l.burst / l.rate * float64(time.Second))
	if now.Sub(l.lastSweep) < idle {
		return
	}
	l.lastSweep = now
	for k, b := range l.buckets {
		if now.Sub(b.last) >= idle {
			delete(l.buckets, k)
		}
	}
}

// size reports how many keys are tracked; used by tests.
func (l *Limiter) size() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.buckets)
}
