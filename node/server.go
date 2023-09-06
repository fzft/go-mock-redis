package node

import (
	"fmt"
	"github.com/fzft/go-mock-redis/db"
	"github.com/fzft/go-mock-redis/log"
	"go.uber.org/zap"
	"net"
	"os"
	"os/signal"
	"syscall"
)

var server *Server

const (
	ProtoIOLen           = 1024 * 16
	ProtoReplyChunkBytes = 16 * 1024
	ProtoInlineMaxSize   = 1024 * 64
	ProtoMBulkBigArg     = 1024 * 32
	ProtoResizeThreshold = 1024 * 32
	ProtoReplyMinBytes   = 1024
	RedisAutoSyncBytes   = 1024 * 1024 * 4 // 512MB
)

const (
	ReplyBufferDefaultPeakResetTime = 5000
)

const MaxFD int64 = 1024

type Server struct {

	// General
	pid            int    // server pid
	configFile     string // Path of config file
	executable     string // Path of executable file
	db             *db.RedisDb
	commands       *db.HashTable[string, RedisCommand]
	originCommands *db.HashTable[string, RedisCommand]
	pidPath        string // pid file path
	reactor        IReactor
	handler        ReaderHandler
	hz             int // serverCron() calls frequency in hertz

	// Networking
	port          int
	tlsPort       int
	bindAddr      []string // Addresses we should bind to
	bindAddrCount int      // Number of addresses in bindAddr
	clients       *db.List[Client]

	// RDB persistence
	dirty uint64 // change to DB from the last save

	// Configuration
	maxIdleTime int64 // default client timeout
	tcpKeepLive int   // default tcp keepalive
	dbNum       int   // default db number

	lastSave int64 // Unix time of last save successful completion

	// logging
	logFile string // Path of log file
}

func NewServer(port int) *Server {
	return &Server{
		port: port,
	}
}

func (s *Server) Run() error {
	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
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

	log.Logger.Info("listening on ", zap.Int("port", s.port))
	reactor.Run()
	log.Logger.Info("shutting down server")
	return nil
}

func (s *Server) SetHandler(handler ReaderHandler) {
	s.handler = handler
}

//func (s *Server) populateCommandTable() {
//	for j := 0; ; j++ {
//		if
//	}
//}
//
//// populateCommandStructure recursively populates the command table starting
//func (s *Server) populateCommandStructure(c RedisCommand) bool {
//	for _, subCmd := range c.SubCommands() {
//
//	}
//}
