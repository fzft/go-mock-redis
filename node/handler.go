package node

import (
	"github.com/fzft/go-mock-redis/log"
	"go.uber.org/zap"
	"io"
)

// ReaderHandler defines an interface for custom read logic.
type ReaderHandler interface {
	Read(conn Conn) error
}

// WriterHandler defines an interface for custom write logic.
type WriterHandler interface {
	Write(conn *Conn, w io.Writer, data []byte) error
}

type ReaderWriterHandler interface {
	ReaderHandler
	WriterHandler
}

// DefaultHandler is a simple implementation of the ReaderHandler.
type DefaultHandler struct{}

func (dh DefaultHandler) Read(conn Conn) error {
	// Default read logic. This can be replaced by the user if they implement their own handler.
	data, err := conn.Read()
	log.Logger.Info("read data", zap.ByteString("data", data))
	if err != nil {
		return err
	}
	conn.Write(data)
	return nil
}

// DefaultWriterHandler is a simple implementation of the WriterHandler.
type DefaultWriterHandler struct{}

func (dwh DefaultWriterHandler) Write(conn *Conn, w io.Writer, data []byte) error {

	// Default write logic. This can be replaced by the user if they implement their own handler.
	_, err := w.Write(data)
	return err
}
