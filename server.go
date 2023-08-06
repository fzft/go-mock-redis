package main

import (
	"go.uber.org/zap"
	"net"
	"os"
	"os/signal"
	"syscall"
)

const MaxFD int64 = 1024

type Server struct {
	addr    string
	reactor *Reactor
	handler ReaderHandler
}

func NewServer(addr string) *Server {
	return &Server{
		addr: addr,
	}
}

func (s *Server) Run() error {
	signals := make(chan os.Signal)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		Logger.Error("listen error: ", zap.Error(err))
		return err
	}

	reactor, err := NewReactor(ln, signals)
	if err != nil {
		return err
	}

	if s.handler == nil {
		s.handler = DefaultHandler{}
	}

	reactor.Handler(s.handler)

	Logger.Info("listening on ", zap.String("addr", s.addr))
	// blocking
	reactor.Run()

	Logger.Info("shutting down server")
	return nil
}

func (s *Server) Handler(handler ReaderHandler) {
	s.handler = handler
}
