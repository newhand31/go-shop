package ratelimit

import (
	"testing"
	"time"
)

type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time          { return c.t }
func (c *fakeClock) advance(d time.Duration) { c.t = c.t.Add(d) }

// countAllowed calls Allow(key) n times and counts the successes.
func countAllowed(l *Limiter, key string, n int) int {
	got := 0
	for i := 0; i < n; i++ {
		if ok, _ := l.Allow(key); ok {
			got++
		}
	}
	return got
}

func TestAllow(t *testing.T) {
	start := time.Date(2026, 11, 11, 20, 31, 0, 0, time.UTC)

	tests := []struct {
		name  string
		rate  float64
		burst int
		run   func(c *fakeClock, l *Limiter) (allowed int)
		want  int
	}{
		{
			name: "burst is allowed then blocked", rate: 1, burst: 3,
			run:  func(_ *fakeClock, l *Limiter) int { return countAllowed(l, "a", 10) },
			want: 3,
		},
		{
			name: "tokens refill over time", rate: 2, burst: 2,
			run: func(c *fakeClock, l *Limiter) int {
				n := countAllowed(l, "a", 5) // drain: 2 allowed
				c.advance(time.Second)       // 2 tokens back
				return n + countAllowed(l, "a", 5)
			},
			want: 4,
		},
		{
			name: "refill never exceeds burst", rate: 100, burst: 2,
			run: func(c *fakeClock, l *Limiter) int {
				l.Allow("a")
				c.advance(time.Hour)
				return countAllowed(l, "a", 10)
			},
			want: 2,
		},
		{
			name: "keys are independent", rate: 1, burst: 1,
			run: func(_ *fakeClock, l *Limiter) int {
				n := 0
				for _, k := range []string{"a", "b", "c", "a", "b"} {
					n += countAllowed(l, k, 1)
				}
				return n
			},
			want: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &fakeClock{t: start}
			l := newWithClock(tt.rate, tt.burst, c.now)
			if got := tt.run(c, l); got != tt.want {
				t.Fatalf("allowed = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestRetryAfter(t *testing.T) {
	c := &fakeClock{t: time.Date(2026, 11, 11, 20, 31, 0, 0, time.UTC)}
	l := newWithClock(2, 1, c.now) // one token every 500ms
	if ok, _ := l.Allow("a"); !ok {
		t.Fatal("first request should pass")
	}
	ok, retry := l.Allow("a")
	if ok {
		t.Fatal("second request should be blocked")
	}
	if retry != 500*time.Millisecond {
		t.Fatalf("retryAfter = %v, want 500ms", retry)
	}
}

func TestSweepDropsIdleKeys(t *testing.T) {
	c := &fakeClock{t: time.Date(2026, 11, 11, 20, 31, 0, 0, time.UTC)}
	l := newWithClock(1, 2, c.now) // a bucket is full again after 2s idle
	for _, k := range []string{"a", "b", "c"} {
		l.Allow(k)
	}
	if got := l.size(); got != 3 {
		t.Fatalf("size = %d, want 3", got)
	}
	c.advance(3 * time.Second)
	l.Allow("d") // any call triggers the sweep
	if got := l.size(); got != 1 {
		t.Fatalf("size after sweep = %d, want 1 (only d)", got)
	}
}
