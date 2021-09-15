package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRateLimitingMiddleware(t *testing.T) {
	t.Parallel()

	const (
		maxRequests     = 5
		handlerRespTime = 100 * time.Millisecond
	)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(handlerRespTime)
	})

	wg := &sync.WaitGroup{}

	middleware := rateLimitingMiddleware(handler, maxRequests)
	for i := 0; i < maxRequests; i++ {
		check(t, wg, middleware, http.StatusOK, "")
	}

	time.Sleep(handlerRespTime / 10)
	for i := 0; i < 10; i++ {
		check(t, wg, middleware, http.StatusTooManyRequests, http.StatusText(http.StatusTooManyRequests))
	}

	time.Sleep(handlerRespTime)
	for i := 0; i < 2*maxRequests; i++ {
		check(t, wg, middleware, http.StatusOK, "")
		time.Sleep(handlerRespTime / 5)
	}

	wg.Wait()
}

func check(t *testing.T, wg *sync.WaitGroup, h http.Handler, wantCode int, wantBody string) {
	t.Helper()

	wg.Add(1)
	go func() {
		defer wg.Done()

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		h.ServeHTTP(w, req)

		if wantCode != w.Code {
			t.Errorf("unexpected status code, want: %v, got: %v", wantCode, w.Code)
		}
		if body := strings.TrimSpace(w.Body.String()); wantBody != body {
			t.Errorf("unexpected body, want: %q, got: %q", wantBody, body)
		}
	}()
}
