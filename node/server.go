package node

import (
	"github.com/fzft/go-mock-redis/log"
	"go.uber.org/zap"
	"net"
	"os"
	"os/signal"
	"syscall"
)

const MaxFD int64 = 1024

type Server struct {
	address string
	reactor *Reactor
	handler ReaderHandler
}

func NewServer(addr string) *Server {
	return &Server{
		address: addr,
	}
}

func (s *Server) Run() error {
	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ln, err := net.Listen("tcp", s.address)
	if err != nil {
		log.Logger.Error("listen error: ", zap.Error(err))
		return err
	}

	reactor, err := NewReactor(ln, sigCh)
	if err != nil {
		return err
	}

	if s.handler == nil {
		s.handler = DefaultHandler{}
	}

	reactor.SetHandler(s.handler)

	log.Logger.Info("listening on ", zap.String("addr", s.address))
	reactor.Run()
	log.Logger.Info("shutting down server")
	return nil
}

func (s *Server) SetHandler(handler ReaderHandler) {
	s.handler = handler
}