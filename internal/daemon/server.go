package daemon

import (
	"context"
	"net"
	"sync"
)

type Server struct {
	addr     string
	listener net.Listener
	mu       sync.Mutex
	router   *Router
}

func NewServer(addr string) *Server {
	return &Server{addr: addr}
}

func NewServerWithRouter(addr string, router *Router) *Server {
	return &Server{addr: addr, router: router}
}

func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener != nil {
		return nil
	}
	ln, err := net.Listen("unix", s.addr)
	if err != nil {
		return err
	}
	s.listener = ln
	go s.acceptLoop(ctx)
	return nil
}

func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener == nil {
		return nil
	}
	err := s.listener.Close()
	s.listener = nil
	return err
}

func (s *Server) acceptLoop(ctx context.Context) {
	s.mu.Lock()
	ln := s.listener
	router := s.router
	s.mu.Unlock()
	if ln == nil {
		return
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		if router == nil {
			_ = conn.Close()
			continue
		}
		go func(c net.Conn) {
			_ = router.ServeConn(ctx, c, "")
			_ = c.Close()
		}(conn)
	}
}
