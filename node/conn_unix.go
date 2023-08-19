//go:build linux
// +build linux

package node

import (
	"bytes"
	"fmt"
	"golang.org/x/sys/unix"
	"os"
)

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

// BufferedConn is the interface for buffered connection.
type BufferedConn interface {
	Conn
	Buffer
}

type DefaultBufferedConn struct {
	fd        int
	ip        string
	outBuffer bytes.Buffer
	poll      *Poll
}

func (c *DefaultBufferedConn) Read() ([]byte, error) {
	// SAFETY: The fd is valid, so this will not return nil.

	var buf bytes.Buffer
	readBuffer := make([]byte, 4096)

	for {
		n, err := unix.Read(c.fd, readBuffer)
		if n > 0 {
			buf.Write(readBuffer[:n])
		}
		if err != nil {
			if IsTemporaryError(err) {
				break
			}
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

func (c *DefaultBufferedConn) Write(data []byte) error {
	// If outBuffer has data, it means previous write(s) didn't succeed fully.
	// Append the new data to the buffer.
	if c.outBuffer.Len() > 0 {
		c.outBuffer.Write(data)
		if err := c.poll.registerWrite(c.fd); err != nil {
			return err
		}
		return nil
	}

	// Try to write the data directly first.
	fdName := fmt.Sprintf("conn_to_%d", c.fd)
	file := os.NewFile(uintptr(c.fd), fdName)
	if file == nil {
		return fmt.Errorf("invalid fd: %d", c.fd)
	}

	n, err := file.Write(data)
	if err != nil {
		// Handle specific error (e.g., EAGAIN or EWOULDBLOCK).
		// If it's one of these errors, data needs to be buffered.
		if IsTemporaryError(err) {
			c.outBuffer.Write(data[n:]) // Only write remaining data to buffer.
			if err = c.poll.registerWrite(c.fd); err != nil {
				return err
			}
		} else {
			return err // Some other error occurred.
		}
	} else if n < len(data) { // Partial write.
		c.outBuffer.Write(data[n:])
		if err = c.poll.registerWrite(c.fd); err != nil {
			return err
		}
	}

	return nil
}

func (c *DefaultBufferedConn) Close() error {
	if err := c.poll.unregister(c.fd); err != nil {
		return err
	}
	return unix.Close(c.fd)
}

// DataToWrite returns the data to write.
func (c *DefaultBufferedConn) DataToWrite() []byte {
	return c.outBuffer.Bytes()
}

// Next moves the buffer forward.
func (c *DefaultBufferedConn) Next(n int) {
	c.outBuffer.Next(n)
}

// Len returns the length of the buffer.
func (c *DefaultBufferedConn) Len() int {
	return c.outBuffer.Len()
}

// Fd returns the file descriptor of the connection.
func (c *DefaultBufferedConn) Fd() int {
	return c.fd
}

// Ip returns the ip of the connection.
func (c *DefaultBufferedConn) Ip() string {
	return c.ip
}
