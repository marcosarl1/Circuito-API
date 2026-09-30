package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"
)

type RateLimiter struct {
	mutex    sync.Mutex
	requests int
	window   time.Duration
	hits     map[string][]time.Time
}

func RateLimit(requests int, window time.Duration) func(http.Handler) http.Handler {
	limiter := &RateLimiter{
		requests: requests,
		window:   window,
		hits:     map[string][]time.Time{},
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if !limiter.allow(clientKey(request), time.Now()) {
				writer.Header().Set("Content-Type", "application/json")
				writer.WriteHeader(http.StatusTooManyRequests)
				_, _ = writer.Write([]byte(`{"detail":"Muitas requisições, tente novamente mais tarde"}`))
				return
			}
			next.ServeHTTP(writer, request)
		})
	}
}

func (limiter *RateLimiter) allow(key string, now time.Time) bool {
	limiter.mutex.Lock()
	defer limiter.mutex.Unlock()

	cutoff := now.Add(-limiter.window)
	valid := limiter.hits[key][:0]
	for _, moment := range limiter.hits[key] {
		if moment.After(cutoff) {
			valid = append(valid, moment)
		}
	}
	if len(valid) >= limiter.requests {
		limiter.hits[key] = valid
		return false
	}
	limiter.hits[key] = append(valid, now)
	return true
}

func clientKey(request *http.Request) string {
	host, _, err := net.SplitHostPort(request.RemoteAddr)
	if err != nil {
		return request.RemoteAddr
	}
	return host
}
