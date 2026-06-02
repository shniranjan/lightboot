// internal/core/ratelimit.go
package core

import (
	"net/http"
	"sync"
	"time"
)

// RateLimiter tracks failed auth attempts per IP and blocks repeat offenders.
type RateLimiter struct {
	mu       sync.Mutex
	attempts map[string]*ipTracker
	// Config
	maxAttempts int           // max failed attempts before block
	window      time.Duration // time window for counting attempts
	blockTime   time.Duration // how long to block after exceeding limit
}

type ipTracker struct {
	failures  int
	firstFail time.Time
	blocked   bool
	blockedAt time.Time
}

// NewRateLimiter creates a rate limiter. Sensible defaults: 10 attempts per
// minute, 5 minute block after exceeding.
func NewRateLimiter(maxAttempts int, window, blockTime time.Duration) *RateLimiter {
	return &RateLimiter{
		attempts:    make(map[string]*ipTracker),
		maxAttempts: maxAttempts,
		window:      window,
		blockTime:   blockTime,
	}
}

// Allow checks if the given request should be allowed. Returns true if the
// request is permitted, false if it should be blocked.
func (rl *RateLimiter) Allow(r *http.Request) bool {
	ip := clientIP(r)
	rl.mu.Lock()
	defer rl.mu.Unlock()

	tracker, exists := rl.attempts[ip]
	now := time.Now()

	if !exists {
		rl.attempts[ip] = &ipTracker{firstFail: now}
		return true
	}

	// Check if IP is blocked
	if tracker.blocked {
		if now.Sub(tracker.blockedAt) < rl.blockTime {
			return false
		}
		// Block expired, reset
		delete(rl.attempts, ip)
		return true
	}

	// Reset if window has passed
	if now.Sub(tracker.firstFail) > rl.window {
		delete(rl.attempts, ip)
		return true
	}

	return true
}

// RecordFailure records a failed authentication attempt.
func (rl *RateLimiter) RecordFailure(r *http.Request) {
	ip := clientIP(r)
	rl.mu.Lock()
	defer rl.mu.Unlock()

	tracker, exists := rl.attempts[ip]
	if !exists {
		rl.attempts[ip] = &ipTracker{failures: 1, firstFail: time.Now()}
		return
	}

	tracker.failures++
	if tracker.failures >= rl.maxAttempts {
		tracker.blocked = true
		tracker.blockedAt = time.Now()
	}
}

// clientIP extracts the client IP from the request, handling X-Forwarded-For
// for reverse proxy setups.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the chain
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}
	// Fall back to RemoteAddr (strip port)
	addr := r.RemoteAddr
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			return addr[:i]
		}
	}
	return addr
}