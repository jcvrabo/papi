package middleware

import (
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type client struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiter is an in-memory token bucket rate limiter keyed by client IP.
type RateLimiter struct {
	clients sync.Map
	rps     rate.Limit
	burst   int
	logger  *slog.Logger
}

// NewRateLimiter creates a new rate limiter with the given requests-per-second and burst size.
func NewRateLimiter(rps float64, burst int, logger *slog.Logger) *RateLimiter {
	rl := &RateLimiter{
		rps:    rate.Limit(rps),
		burst:  burst,
		logger: logger,
	}
	go rl.cleanup()
	return rl
}

// Handler returns an HTTP middleware that enforces rate limiting.
func (rl *RateLimiter) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, _ := net.SplitHostPort(r.RemoteAddr)
		if ip == "" {
			ip = r.RemoteAddr
		}

		limiter := rl.getLimiter(ip)
		if !limiter.Allow() {
			rl.logger.Warn("rate limit exceeded", "ip", ip)
			http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) getLimiter(ip string) *rate.Limiter {
	if v, ok := rl.clients.Load(ip); ok {
		c := v.(*client)
		c.lastSeen = time.Now()
		return c.limiter
	}

	limiter := rate.NewLimiter(rl.rps, rl.burst)
	rl.clients.Store(ip, &client{limiter: limiter, lastSeen: time.Now()})
	return limiter
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.clients.Range(func(key, value any) bool {
			c := value.(*client)
			if time.Since(c.lastSeen) > 3*time.Minute {
				rl.clients.Delete(key)
			}
			return true
		})
	}
}
