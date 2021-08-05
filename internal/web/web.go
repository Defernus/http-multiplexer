package web

import (
	"context"
	"log"
	"net/http"
	"time"
)

type Server struct {
	httpServer *http.Server
	cfg        Config
	shutdown   chan struct{}
}

type Config struct {
	Addr string
}

func NewServer(
	cfg Config,
) (*Server, error) {
	s := &Server{
		cfg:        cfg,
		httpServer: nil,
		shutdown:   make(chan struct{}),
	}

	s.httpServer = &http.Server{
		Addr:    cfg.Addr,
		Handler: s,
	}

	return s, nil
}

func (s *Server) Start() error {
	log.Printf("start server at http://0.0.0.0%v", s.cfg.Addr)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) Shutdown() error {
	close(s.shutdown)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return s.httpServer.Shutdown(ctx)
}
