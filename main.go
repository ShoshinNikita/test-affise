package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ShoshinNikita/test-affise/internal/server"
)

func main() {
	var (
		serverPort     = flag.Int("port", 8080, "server port")
		workerCount    = flag.Int("worker-count", 4, "number of workers that fetch urls")
		requestTimeout = flag.Duration("req-timeout", time.Second, "request timeout")
	)
	flag.Parse()

	// Start server
	s := server.NewServer(server.Config{
		Port:           *serverPort,
		WorkerCount:    *workerCount,
		RequestTimeout: *requestTimeout,
	})
	errCh := make(chan error, 1)
	go func() {
		if err := s.Start(); err != nil {
			errCh <- err
		}
	}()

	termCh := make(chan os.Signal, 1)
	signal.Notify(termCh, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-termCh:
		log.Println("[INF] got an interrupt signal")
	case err := <-errCh:
		log.Printf("[ERR] server error, stop app: %s", err)
	}

	// Shutdown components
	log.Println("[INF] stop app...")

	if err := s.Stop(); err != nil {
		log.Printf("[ERR] couldn't gracefully shutdown server: %s", err)
	}
}
