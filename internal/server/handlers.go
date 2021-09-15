package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"
)

func handleFetchURLs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	const maxURLsCount = 20

	var req struct {
		URLs []string `json:"urls"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("couldn't decode request: %s", err), http.StatusBadRequest)
		return
	}
	if len(req.URLs) > maxURLsCount {
		http.Error(w, fmt.Sprintf("max number of urls is %d", maxURLsCount), http.StatusBadRequest)
		return
	}
	for _, rawURL := range req.URLs {
		if _, err := url.Parse(rawURL); err != nil {
			http.Error(w, fmt.Sprintf("url %q is invalid", rawURL), http.StatusBadRequest)
			return
		}
	}

	res, err := fetchURLs(r.Context(), req.URLs)
	if err != nil {
		http.Error(w, fmt.Sprintf("url fetching failed: %s", err), http.StatusInternalServerError)
		return
	}

	resp := make(map[string]string, len(res))
	for _, r := range res {
		resp[r.URL] = r.Body
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("[ERR] couldn't encode response: %s", err)
	}
}

func fetchURLs(ctx context.Context, urls []string) (res []FetchResult, firstErr error) {
	const maxWorkerCount = 4

	urlsCh := make(chan string, maxWorkerCount)
	go func() {
		for _, u := range urls {
			urlsCh <- u
		}
		close(urlsCh)
	}()
	defer func() {
		// Sending to the channel can be blocked if workers are stopped because of an error.
		// So, drain the channel to stop the goroutine
		for range urlsCh {
		}
	}()

	ctx, cancel := context.WithCancel(ctx)

	var (
		errCh             = make(chan error)
		errProcessingDone = make(chan struct{})
	)
	go func() {
		for err := range errCh {
			select {
			case <-ctx.Done():
			default:
				firstErr = err // return the first error
			}
			cancel()

			log.Printf("[ERR] couldn't fetch url: %s", err)
		}
		close(errProcessingDone)
	}()

	var (
		workerResults FetchResults
		wg            sync.WaitGroup
	)
	for i := 0; i < maxWorkerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			res, err := startFetchURLWorker(ctx, urlsCh)
			if err != nil {
				errCh <- err
				return
			}
			workerResults.Append(res...)
		}()
	}
	wg.Wait()
	close(errCh)

	<-errProcessingDone

	if firstErr != nil {
		return nil, firstErr
	}
	return workerResults.Get(), nil
}

func startFetchURLWorker(ctx context.Context, urlsCh <-chan string) (res []FetchResult, err error) {
	for {
		select {
		case <-ctx.Done():
			return nil, nil

		case u, ok := <-urlsCh:
			if !ok {
				// No more urls
				return res, nil
			}

			body, err := fetchURL(ctx, u)
			if err != nil {
				return nil, fmt.Errorf("couldn't fetch %q: %w", u, err)
			}
			res = append(res, FetchResult{
				URL:  u,
				Body: body,
			})
		}
	}
}

func fetchURL(ctx context.Context, url string) (body string, err error) {
	const requestTimeout = time.Second

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("couldn't build request: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("couldn't read body: %w", err)
	}
	return string(b), nil
}

type FetchResult struct {
	URL  string
	Body string
}

type FetchResults struct {
	mu  sync.Mutex
	res []FetchResult
}

func (res *FetchResults) Append(s ...FetchResult) {
	res.mu.Lock()
	defer res.mu.Unlock()

	res.res = append(res.res, s...)
}

func (res *FetchResults) Get() []FetchResult {
	res.mu.Lock()
	defer res.mu.Unlock()

	return res.res
}
