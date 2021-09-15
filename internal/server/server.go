package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"
)

const maxConcurrentRequests = 100

type Server struct {
	cfg        Config
	httpServer *http.Server
}

type Config struct {
	Port           int
	WorkerCount    int
	RequestTimeout time.Duration
}

func NewServer(cfg Config) *Server {
	s := &Server{
		cfg: cfg,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleFetchURLs)

	handler := rateLimitingMiddleware(mux, maxConcurrentRequests)

	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: handler,
	}
	return s
}

func (s *Server) Start() error {
	log.Printf("[DBG] start server on port :%d", s.cfg.Port)

	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return s.httpServer.Shutdown(ctx)
}
