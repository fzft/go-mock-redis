package node

// Conn is the interface for connection.
type Conn interface {
	// Read reads data from the connection.
	Read() (data []byte, err error)

	// Write writes data to the connection.
	Write(data []byte) (err error)

	// Close closes the connection.
	Close() error

	Fd() int
	Ip() string
}

type Buffer interface {
	// DataToWrite closes the connection.
	DataToWrite() []byte

	Next(n int)

	Len() int
}

// BufferedConn is the interface for buffered connection.
type BufferedConn interface {
	Conn
	Buffer
}
