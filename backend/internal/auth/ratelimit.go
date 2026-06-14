package auth

import (
	"sync"
	"time"
)

// Limiter is a simple in-memory failure-backoff tracker keyed by an arbitrary
// string (e.g. account email or client IP). After max consecutive failures for a
// key it blocks further attempts for cooldown; a success resets the key.
//
// In-memory is adequate for MVP single-instance deploys. A multi-instance
// deployment should back this with a shared store (Redis); the interface stays.
type Limiter struct {
	mu       sync.Mutex
	entries  map[string]*limEntry
	max      int
	cooldown time.Duration
	now      func() time.Time
}

type limEntry struct {
	failures     int
	blockedUntil time.Time
}

// NewLimiter returns a Limiter that blocks for cooldown after max failures. now
// defaults to time.Now (UTC) when nil.
func NewLimiter(max int, cooldown time.Duration, now func() time.Time) *Limiter {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &Limiter{
		entries:  make(map[string]*limEntry),
		max:      max,
		cooldown: cooldown,
		now:      now,
	}
}

// Retry reports whether key is currently blocked and, if so, how long until it
// may try again.
func (l *Limiter) Retry(key string) (blocked bool, retryAfter time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	e, ok := l.entries[key]
	if !ok {
		return false, 0
	}
	now := l.now()
	if now.Before(e.blockedUntil) {
		return true, e.blockedUntil.Sub(now)
	}
	return false, 0
}

// Fail records a failure for key, arming the block once max is reached.
func (l *Limiter) Fail(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	e, ok := l.entries[key]
	if !ok {
		e = &limEntry{}
		l.entries[key] = e
	}
	e.failures++
	if e.failures >= l.max {
		e.blockedUntil = l.now().Add(l.cooldown)
		e.failures = 0 // window resets after the cooldown elapses
	}
}

// Reset clears any failure state for key (call on a successful attempt).
func (l *Limiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.entries, key)
}
