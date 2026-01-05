package server

import (
	"net/http"

	"golang.org/x/time/rate"
)

// CORSMiddleware adds CORS headers to responses.
// If origins is empty or nil, CORS headers are not added.
func CORSMiddleware(origins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if len(origins) == 0 {
			return next
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			allowed := false
			for _, o := range origins {
				if o == "*" || o == origin {
					allowed = true
					break
				}
			}

			if allowed && origin != "" {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				w.Header().Set("Access-Control-Max-Age", "86400") // 24 hours
			}

			// Handle preflight request
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RateLimitMiddleware limits requests using a token bucket algorithm.
// rps is the rate limit in requests per second, burst is the maximum burst size.
func RateLimitMiddleware(rps float64, burst int) func(http.Handler) http.Handler {
	limiter := rate.NewLimiter(rate.Limit(rps), burst)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.Allow() {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", "1")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error": "rate limit exceeded"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
