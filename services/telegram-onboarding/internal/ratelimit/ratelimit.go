// Package ratelimit provides a per-client token bucket.
package ratelimit

import (
	"sync"
	"time"
)

type bucket struct {
	tokens   float64
	lastSeen time.Time
}

// Limiter allows burst requests per key, refilling at rate per second.
type Limiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket

	burst float64
	rate  float64
	idle  time.Duration
	now   func() time.Time
}

// New returns a limiter allowing burst requests immediately, refilling at
// rate tokens per second.
func New(burst int, rate float64) *Limiter {
	return &Limiter{
		buckets: make(map[string]*bucket),
		burst:   float64(burst),
		rate:    rate,
		idle:    10 * time.Minute,
		now:     time.Now,
	}
}

// SetClock replaces the limiter's clock. For tests.
func (l *Limiter) SetClock(now func() time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.now = now
}

// Allow consumes one token for key, reporting whether the request may proceed.
func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.sweepLocked(now)

	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.burst, lastSeen: now}
		l.buckets[key] = b
	} else {
		b.tokens += now.Sub(b.lastSeen).Seconds() * l.rate
		if b.tokens > l.burst {
			b.tokens = l.burst
		}
		b.lastSeen = now
	}

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

func (l *Limiter) sweepLocked(now time.Time) {
	for key, b := range l.buckets {
		if now.Sub(b.lastSeen) > l.idle {
			delete(l.buckets, key)
		}
	}
}
