package core

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// AuthMiddleware wraps the given http.Handler with Bearer token authentication.
// Routes with a matching path prefix are protected; all others pass through.
// This uses the API token from the Config.
func AuthMiddleware(next http.Handler, cfg *Config, rl *RateLimiter, publicPrefixes ...string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Check if this path is public
		for _, prefix := range publicPrefixes {
			if strings.HasPrefix(path, prefix) {
				next.ServeHTTP(w, r)
				return
			}
		}

		// Also allow /api/health and the /boot/* chain endpoints
		if path == "/api/health" || strings.HasPrefix(path, "/api/boot/") || strings.HasPrefix(path, "/cache/") {
			next.ServeHTTP(w, r)
			return
		}

		// In AuthMiddleware, before the token check:
		// Check rate limit
		if rl != nil && !rl.Allow(r) {
			http.Error(w, `{"error":"too many requests, try again later"}`, http.StatusTooManyRequests)
			return
		}

		// Verify Bearer token for everything else
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
			return
		}
		token := strings.TrimPrefix(auth, "Bearer ")

		if subtle.ConstantTimeCompare([]byte(token), []byte(cfg.GetAPIToken())) != 1 {
			if rl != nil {
				rl.RecordFailure(r)
			}
			http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
			return
		}


		next.ServeHTTP(w, r)
	})
}

