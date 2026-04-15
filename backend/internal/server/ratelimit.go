package server

import (
	"net/http"
	"sync"
	"time"
)

// 简易内存级限流：按 IP 每分钟限制请求数。fpk 单机足够用。
type rateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	max     int
}

type bucket struct {
	count int
	reset time.Time
}

func newRateLimiter(maxPerMin int) *rateLimiter {
	return &rateLimiter{buckets: make(map[string]*bucket), max: maxPerMin}
}

func (rl *rateLimiter) allow(key string) bool {
	if rl.max <= 0 {
		return true
	}
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	b, ok := rl.buckets[key]
	if !ok || now.After(b.reset) {
		rl.buckets[key] = &bucket{count: 1, reset: now.Add(time.Minute)}
		return true
	}
	if b.count >= rl.max {
		return false
	}
	b.count++
	return true
}

func (s *Server) limitUpload(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r, s.cfg.Server.TrustProxy)
		if !s.rate.allow(ip) {
			w.Header().Set("Retry-After", "60")
			writeErr(w, http.StatusTooManyRequests, "上传过于频繁，请稍后再试")
			return
		}
		next(w, r)
	}
}
