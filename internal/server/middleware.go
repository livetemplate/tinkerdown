package server

import (
	"container/list"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/livetemplate/tinkerdown/internal/config"

	"golang.org/x/time/rate"
)

// CORSMiddleware adds CORS headers to responses.
// If origins is empty or nil, CORS headers are not added.
// authHeaderName is included in Access-Control-Allow-Headers when non-empty.
func CORSMiddleware(origins []string, authHeaderName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if len(origins) == 0 {
			return next
		}

		// Build allowed headers list, including the configured auth header
		allowHeaders := "Content-Type, Authorization, X-API-Key"
		if authHeaderName != "" && authHeaderName != "Authorization" && authHeaderName != "X-API-Key" {
			allowHeaders += ", " + authHeaderName
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			allowed := false
			allowAll := false
			for _, o := range origins {
				if o == "*" {
					allowed = true
					allowAll = true
					break
				}
				if o == origin {
					allowed = true
					break
				}
			}

			if allowed && origin != "" {
				// When wildcard is configured, use "*" header; otherwise echo the specific origin
				if allowAll {
					w.Header().Set("Access-Control-Allow-Origin", "*")
				} else {
					w.Header().Set("Access-Control-Allow-Origin", origin)
				}
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", allowHeaders)
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

// SecurityHeadersMiddleware adds security headers to all responses.
func SecurityHeadersMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			// CSP: allow self, inline styles (needed for PicoCSS and LVT rendering),
			// unsafe-eval (needed for Mermaid.js), and data: URIs for fonts/images.
			// connect-src 'self' covers same-origin WebSocket (ws/wss) connections.
			w.Header().Set("Content-Security-Policy",
				"default-src 'self'; "+
					"script-src 'self' 'unsafe-inline' 'unsafe-eval'; "+
					"style-src 'self' 'unsafe-inline'; "+
					"img-src 'self' data: https:; "+
					"font-src 'self' data:; "+
					"connect-src 'self'; "+
					"frame-ancestors 'none'")

			next.ServeHTTP(w, r)
		})
	}
}

// evictionLogInterval is the minimum time between eviction log messages.
const evictionLogInterval = 30 * time.Second

// ipLimiter tracks a per-IP token bucket and its position in the LRU list.
type ipLimiter struct {
	ip       string
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimitMiddleware limits requests using a token bucket algorithm with per-IP tracking.
// rps is the rate limit in requests per second, burst is the maximum burst size,
// and maxIPs is the maximum number of unique IPs to track (LRU eviction when full).
//
// The cleanup goroutine starts immediately when this function is called.
// The ctx parameter controls its lifetime; cancel ctx to stop it.
// The returned channel is closed when the goroutine exits,
// allowing callers to wait for a clean shutdown.
func RateLimitMiddleware(ctx context.Context, rps float64, burst int, maxIPs int) (func(http.Handler) http.Handler, <-chan struct{}) {
	if maxIPs <= 0 {
		maxIPs = 10000
	}

	var (
		items = make(map[string]*list.Element)
		order = list.New() // front = most recent, back = oldest
		mu    sync.Mutex

		// Eviction logging state (always accessed under mu)
		lastEvictLog time.Time
		evictCount   int
	)

	// Start cleanup goroutine — exits when ctx is cancelled.
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				mu.Lock()
				now := time.Now()
				// Iterate all entries and remove stale ones.
				// Cannot break early: LRU order tracks access recency,
				// not lastSeen time, so stale entries may appear anywhere.
				for e := order.Back(); e != nil; {
					lim := e.Value.(*ipLimiter)
					prev := e.Prev()
					if now.Sub(lim.lastSeen) > 10*time.Minute {
						order.Remove(e)
						delete(items, lim.ip)
					}
					e = prev
				}
				mu.Unlock()
			case <-ctx.Done():
				return
			}
		}
	}()

	middleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := getClientIP(r)

			mu.Lock()
			elem, exists := items[ip]
			if exists {
				// Move to front (most recently used) and update lastSeen
				order.MoveToFront(elem)
				elem.Value.(*ipLimiter).lastSeen = time.Now()
			} else {
				// Evict least recently used if at capacity
				if order.Len() >= maxIPs {
					back := order.Back()
					if back != nil {
						evicted := back.Value.(*ipLimiter)
						order.Remove(back)
						delete(items, evicted.ip)
						evictCount++
						if time.Since(lastEvictLog) >= evictionLogInterval {
							log.Printf("[RateLimit] Evicted %d least-recent IP(s) (at capacity: %d IPs)", evictCount, maxIPs)
							lastEvictLog = time.Now()
							evictCount = 0
						}
					}
				}
				lim := &ipLimiter{
					ip:       ip,
					limiter:  rate.NewLimiter(rate.Limit(rps), burst),
					lastSeen: time.Now(),
				}
				elem = order.PushFront(lim)
				items[ip] = elem
			}
			allowed := elem.Value.(*ipLimiter).limiter.Allow()
			mu.Unlock()

			if !allowed {
				w.Header().Set("Retry-After", "1")
				writeJSONError(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}
			next.ServeHTTP(w, r)
		})
	}

	return middleware, done
}

// getClientIP extracts the client IP from the request.
// It only trusts X-Forwarded-For / X-Real-IP when the immediate peer is a
// loopback or private address (i.e., behind a reverse proxy).
func getClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}

	peerIP := net.ParseIP(host)
	trustedProxy := peerIP != nil && (peerIP.IsLoopback() || peerIP.IsPrivate())

	if trustedProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			if parts := strings.SplitN(xff, ",", 2); len(parts) > 0 {
				return strings.TrimSpace(parts[0])
			}
		}
		if xri := r.Header.Get("X-Real-IP"); xri != "" {
			return strings.TrimSpace(xri)
		}
	}

	if peerIP != nil {
		return peerIP.String()
	}
	return host
}

// authContextKey is the type for auth-related context keys.
type authContextKey string

const (
	ctxKeyName        authContextKey = "auth_key_name"
	ctxKeyPermissions authContextKey = "auth_permissions"
)

// AuthMiddleware validates API key authentication with support for multiple keys.
// If no keys are configured, authentication is disabled and all requests pass through.
func AuthMiddleware(authCfg *config.AuthConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if authCfg == nil {
			return next
		}

		apiKeys := authCfg.GetAPIKeys()
		if len(apiKeys) == 0 {
			return next
		}

		headerName := authCfg.GetHeaderName()

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := r.Header.Get(headerName)
			if token == "" {
				writeJSONError(w, http.StatusUnauthorized, "authentication required")
				return
			}

			// Handle "Authorization: Bearer <token>" format
			if headerName == "Authorization" {
				const bearerPrefix = "Bearer "
				if len(token) > len(bearerPrefix) && token[:len(bearerPrefix)] == bearerPrefix {
					token = token[len(bearerPrefix):]
				} else {
					writeJSONError(w, http.StatusUnauthorized, "invalid authorization format, expected Bearer token")
					return
				}
			}

			// Find matching key
			var matched *config.APIKeyConfig
			for i := range apiKeys {
				expandedKey := os.ExpandEnv(apiKeys[i].Key)
				if secureCompare(token, expandedKey) {
					matched = &apiKeys[i]
					break
				}
			}

			if matched == nil {
				writeJSONError(w, http.StatusUnauthorized, "invalid API key")
				return
			}

			// Set auth context
			ctx := context.WithValue(r.Context(), ctxKeyName, matched.Name)
			ctx = context.WithValue(ctx, ctxKeyPermissions, matched.Permissions)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// MethodPermissionMiddleware checks that the authenticated key has permission
// for the requested HTTP method. Must be used after AuthMiddleware.
func MethodPermissionMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// OPTIONS bypass (CORS preflight)
			if r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			var required config.Permission
			switch r.Method {
			case http.MethodGet, http.MethodHead:
				required = config.PermRead
			case http.MethodPost, http.MethodPut, http.MethodPatch:
				required = config.PermWrite
			case http.MethodDelete:
				required = config.PermDelete
			default:
				writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
				return
			}

			if !HasPermission(r, required) {
				writeJSONError(w, http.StatusForbidden,
					fmt.Sprintf("insufficient permissions: %s required for %s", required, r.Method))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// HasPermission checks if the request has a specific permission from auth context.
func HasPermission(r *http.Request, perm config.Permission) bool {
	perms, ok := r.Context().Value(ctxKeyPermissions).([]config.Permission)
	if !ok {
		return false
	}
	for _, p := range perms {
		if p == perm {
			return true
		}
	}
	return false
}

// secureCompare performs a constant-time string comparison to prevent timing attacks.
func secureCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var result byte
	for i := 0; i < len(a); i++ {
		result |= a[i] ^ b[i]
	}
	return result == 0
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
