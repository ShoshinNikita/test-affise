package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"sort"
	"testing"
	"time"
)

func TestFetchURLs(t *testing.T) {
	t.Parallel()

	const (
		respTime   = 100 * time.Millisecond
		reqTimeout = time.Second
	)

	addr := startServer(t, respTime)
	urls := buildURLs(addr, "/1", "/2", "/3", "/4", "/5")
	want := buildFetchResults(urls...)

	res, err := fetchURLs(context.Background(), 3, reqTimeout, urls)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	checkResults(t, want, res)
}

func TestFetchURLs_Timeout(t *testing.T) {
	t.Parallel()

	const (
		respTime   = 100 * time.Millisecond
		reqTimeout = 50 * time.Millisecond
	)

	addr := startServer(t, respTime)
	urls := buildURLs(addr, "/1", "/2", "/3", "/4", "/5")

	_, err := fetchURLs(context.Background(), 3, reqTimeout, urls)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("unexpected error, want: %v, got: %v", context.DeadlineExceeded, err)
	}
}

func TestFetchURLs_InvalidURL(t *testing.T) {
	t.Parallel()

	const (
		respTime   = 100 * time.Millisecond
		reqTimeout = time.Second
	)

	addr := startServer(t, respTime)
	urls := []string{"abc" + addr + "/1"}

	_, err := fetchURLs(context.Background(), 3, reqTimeout, urls)
	if err == nil {
		t.Fatalf("want error, got <nil>")
	}
}

func startServer(t *testing.T, respTime time.Duration) (addr string) {
	t.Helper()

	port, err := getFreePort()
	if err != nil {
		t.Errorf("couldn't get free port: %s", err)
	}

	server := http.Server{
		Addr: fmt.Sprintf(":%d", port),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(respTime)
			w.Write([]byte("http://" + r.Host + r.URL.String())) //nolint:errcheck
		}),
	}
	t.Cleanup(func() {
		if err := server.Shutdown(context.Background()); err != nil {
			t.Errorf("couldn't shutdown server: %s", err)
		}
	})

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("got unexpected error from server: %s", err)
		}
	}()

	return fmt.Sprintf("http://localhost:%d", port)
}

func getFreePort() (int, error) {
	listener, err := net.Listen("tcp", "")
	if err != nil {
		return 0, err
	}
	defer listener.Close()

	tcpAddr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return 0, fmt.Errorf("listener.Addr() has unexpected type: %T", listener.Addr())
	}

	return tcpAddr.Port, nil
}

func buildURLs(addr string, paths ...string) []string {
	res := make([]string, 0, len(paths))
	for _, u := range paths {
		res = append(res, addr+u)
	}
	return res
}

func buildFetchResults(urls ...string) []FetchResult {
	res := make([]FetchResult, 0, len(urls))
	for _, u := range urls {
		res = append(res, FetchResult{
			URL:  u,
			Body: u,
		})
	}
	return res
}

func checkResults(t *testing.T, want, got []FetchResult) {
	t.Helper()

	sort := func(s []FetchResult) {
		sort.Slice(s, func(i, j int) bool {
			return s[i].URL < s[j].URL
		})
	}
	sort(got)
	sort(want)

	if !reflect.DeepEqual(want, got) {
		t.Errorf("unexpected fetch results\n\twant: %+v\n\tgot: %+v", want, got)
	}
}
