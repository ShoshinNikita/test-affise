package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"
)

type Server struct {
	cfg        Config
	httpServer *http.Server
}

type Config struct {
	Port int
}

func NewServer(cfg Config) *Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handleFetchURLs)

	return &Server{
		cfg: cfg,
		httpServer: &http.Server{
			Addr:    fmt.Sprintf(":%d", cfg.Port),
			Handler: mux,
		},
	}
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
