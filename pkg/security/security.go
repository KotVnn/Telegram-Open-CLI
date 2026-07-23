package security

import (
	"net/http"
	"sync"
	"time"
)

// RateLimiter provides rate limiting for HTTP requests.
type RateLimiter struct {
	mu       sync.Mutex
	clients  map[string]*clientInfo
	limit    int
	window   time.Duration
	stopCh   chan struct{}
}

type clientInfo struct {
	count    int
	lastSeen time.Time
}

// NewRateLimiter creates a new rate limiter.
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		clients: make(map[string]*clientInfo),
		limit:   limit,
		window:  window,
		stopCh:  make(chan struct{}),
	}

	// Cleanup old entries periodically
	go func() {
		ticker := time.NewTicker(window)
		defer ticker.Stop()

		for {
			select {
			case <-rl.stopCh:
				return
			case <-ticker.C:
				rl.cleanup()
			}
		}
	}()

	return rl
}

// Stop stops the cleanup goroutine.
func (rl *RateLimiter) Stop() {
	close(rl.stopCh)
}

// Allow checks if a request from the given IP is allowed.
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	client, exists := rl.clients[ip]

	if !exists {
		rl.clients[ip] = &clientInfo{
			count:    1,
			lastSeen: now,
		}
		return true
	}

	// Reset if window has passed
	if now.Sub(client.lastSeen) > rl.window {
		client.count = 1
		client.lastSeen = now
		return true
	}

	if client.count >= rl.limit {
		return false
	}

	client.count++
	client.lastSeen = now
	return true
}

// cleanup removes old client entries.
func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for ip, client := range rl.clients {
		if now.Sub(client.lastSeen) > rl.window*2 {
			delete(rl.clients, ip)
		}
	}
}

// Middleware returns an HTTP middleware that applies rate limiting.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			ip = forwarded
		}

		if !rl.Allow(ip) {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// ValidateInput validates user input to prevent injection attacks.
func ValidateInput(input string, maxLength int) bool {
	if len(input) > maxLength {
		return false
	}

	// Check for null bytes
	for _, b := range []byte(input) {
		if b == 0 {
			return false
		}
	}

	return true
}

// SanitizeInput removes control characters while preserving valid UTF-8.
func SanitizeInput(input string) string {
	result := make([]rune, 0, len(input))
	for _, r := range input {
		// Allow printable characters (including Unicode)
		// Block: null, control chars (0-31), DEL (127), C1 control (128-159)
		if r == 0 || (r >= 1 && r <= 31) || r == 127 || (r >= 128 && r <= 159) {
			continue
		}
		result = append(result, r)
	}
	return string(result)
}
