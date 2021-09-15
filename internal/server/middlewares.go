package server

import (
	"net/http"
	"sync/atomic"
)

func rateLimitingMiddleware(h http.Handler, maxConcurrentRequests int64) http.Handler {
	var reqCount int64

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		curr := atomic.AddInt64(&reqCount, 1)
		defer atomic.AddInt64(&reqCount, -1)

		if curr > maxConcurrentRequests {
			http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			return
		}

		h.ServeHTTP(w, r)
	})
}
