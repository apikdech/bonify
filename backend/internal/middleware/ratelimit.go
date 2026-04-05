package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimit creates a rate limiting middleware using Redis
// limit is the number of requests allowed per window
// window is the time duration for the rate limit window
func RateLimit(redisClient *redis.Client, limit int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get user ID from context (set by JWT middleware)
			userID := GetUserID(r.Context())

			// If no user ID, fall back to IP address
			var key string
			if userID.String() != "" && userID.String() != "00000000-0000-0000-0000-000000000000" {
				key = fmt.Sprintf("ratelimit:user:%s", userID.String())
			} else {
				key = fmt.Sprintf("ratelimit:ip:%s", r.RemoteAddr)
			}

			// Use Redis to track requests
			ctx := r.Context()

			// Increment the counter
			pipe := redisClient.Pipeline()
			incr := pipe.Incr(ctx, key)
			pipe.Expire(ctx, key, window)
			_, err := pipe.Exec(ctx)

			if err != nil {
				// If Redis is unavailable, allow the request (fail open)
				next.ServeHTTP(w, r)
				return
			}

			count := incr.Val()

			// Set rate limit headers
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", max(0, limit-int(count))))

			// Check if limit exceeded
			if int(count) > limit {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error": "rate limit exceeded"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
