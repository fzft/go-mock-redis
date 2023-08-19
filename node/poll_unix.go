//go:build linux
// +build linux

package node

import (
	"fmt"
	"github.com/fzft/go-mock-redis/log"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"
	"net"
	"sync/atomic"
	"unsafe"
)

// https://copyconstruct.medium.com/the-method-to-epolls-madness-d9d2d6378642

const (
	readEvents      = unix.EPOLLPRI | unix.EPOLLIN
	writeEvents     = unix.EPOLLOUT
	readWriteEvents = readEvents | writeEvents
)

type pipeSignal uint64

const (
	SignalStop pipeSignal = 1
)

func NewPoll(done chan struct{}, size int64, lnFd int) (*Poll, error) {
	// Create a new epoll instance
	epfd, err := unix.EpollCreate1(unix.EPOLL_CLOEXEC)
	if err != nil {
		log.Logger.Error("Failed to create epoll", zap.Error(err))
		return nil, err
	}

	r := NewRegistry(epfd)

	efd, err := unix.Eventfd(0, unix.EFD_NONBLOCK|unix.EFD_CLOEXEC)
	if err != nil {
		log.Logger.Error("Failed to create eventfd", zap.Error(err))
		return nil, err
	}

	// Register the eventfd to epoll for read events
	if err := r.AddRead(efd); err != nil {
		log.Logger.Error("Failed to add eventfd to epoll", zap.Error(err))
		return nil, err
	}

	// Register the listener to epoll for read events
	if err := r.AddRead(lnFd); err != nil {
		log.Logger.Error("Failed to add listener to epoll", zap.Error(err))
		return nil, err
	}

	poll := &Poll{
		Registry: r,
		epollFd:  epfd,
		listenFD: lnFd,
		maxFD:    size,
		connPool: make(map[int]BufferedConn),
		done:     done,
		efd:      efd,
	}

	return poll, nil
}

// CloseGracefully order: eventfd, listener, connections, epoll
// prevent the fd leak
func (p *Poll) CloseGracefully() error {

	// close the eventfd fd
	if err := p.Delete(p.efd); err != nil {
		log.Logger.Debug("Failed to delete eventfd from epoll", zap.Error(err))
	}

	if err := CloseFd(p.efd); err != nil {
		log.Logger.Debug("Failed to close eventfd", zap.Error(err))
	}

	// close the listener fd
	if err := p.Delete(p.listenFD); err != nil {
		log.Logger.Debug("Failed to delete listener from epoll", zap.Error(err))
	}

	if err := CloseFd(p.listenFD); err != nil {
		log.Logger.Debug("Failed to close listener", zap.Error(err))
	}

	// close all connections
	if err := p.ClosAndClearAllFDs(); err != nil {
		log.Logger.Debug("Failed to close connections", zap.Error(err))
	}

	// close the epoll fd
	if err := CloseFd(p.epollFd); err != nil {
		log.Logger.Info("Failed to close epoll", zap.Error(err))
	}

	return nil
}

func (p *Poll) poll() {
	events := make([]unix.EpollEvent, p.maxFD)
	msec := -1

	defer close(p.done)

	// handle cleanup if necessary,
	defer p.CloseGracefully()

	for {
		// EpollWait blocks until there is an event to report
		// n: number of events returned
		// if n ==0 , it means that the call timed out and no events were available
		// if n < 0, it means that an error occurred
		// level triggered, poll mode
		n, err := unix.EpollWait(p.epollFd, events, msec)
		if n == 0 || (n < 0 &&
			err == unix.EINTR) {
			log.Logger.Warn("epoll wait timeout")
			continue
		} else if err != nil {
			log.Logger.Error("epoll wait error", zap.Error(err))
			return
		}

		for i := 0; i < n; i++ {
			ev := &events[i]
			err := p.processEvent(int(ev.Fd), ev)
			switch err {
			case nil:
			case ErrSignalStopped:
				return
			default:
				log.Logger.Error("Failed to process event", zap.Error(err))
				return
			}
		}
	}
}

func (p *Poll) processEvent(fd int, ev *unix.EpollEvent) error {
	if ev.Events&unix.EPOLLERR != 0 || ev.Events&unix.EPOLLHUP != 0 {
		log.Logger.Debug("epoll error event for fd ", zap.Int("fd", fd))

		p.decrFd()

		// remove the fd from epoll set
		return p.unregister(fd)
	}

	if fd == p.efd {
		// if the fd is the read end of the eventfd, it means that there is a signal to handle
		return p.handleSignal(fd)
	} else if fd == p.listenFD {
		// if the fd is the listener, it means that there is a new connection
		return p.accept(fd)
	} else {
		// if the fd is not the listener, it means that there is data to read or write
		if ev.Events&unix.EPOLLIN != 0 {
			conn, ok := p.connPool[fd]
			if !ok {
				log.Logger.Error("connection not found")
				return fmt.Errorf("connection not found for fd %d", fd)
			}
			return p.rHandler.Read(conn)
		}

		if ev.Events&unix.EPOLLOUT != 0 {

			conn, ok := p.connPool[fd]
			if !ok {
				log.Logger.Error("connection not found")
				return fmt.Errorf("connection not found for fd %d", fd)
			}

			return p.handleWrite(conn)
		}
	}
	return nil
}

// handleSignal handles the signal from the signal pipe
func (p *Poll) handleSignal(fd int) error {
	var buf uint64
	_, err := unix.Read(fd, (*(*[8]byte)(unsafe.Pointer(&buf)))[:])
	if err != nil {
		log.Logger.Error("Failed to read from event fd", zap.Error(err))
		return nil
	}
	receivedSignal := pipeSignal(buf)
	switch receivedSignal {
	case SignalStop:
		return ErrSignalStopped
	}
	return nil
}

// sendSignal sends a signal to the event fd
func (p *Poll) sendSignal(sig pipeSignal) error {
	_, err := unix.Write(p.efd, (*(*[8]byte)(unsafe.Pointer(&sig)))[:])
	if err != nil {
		log.Logger.Error("Failed to write to event fd", zap.Error(err))
	}
	return err
}

// accept a new connection
func (p *Poll) accept(fd int) error {
	connFd, sa, err := unix.Accept(fd)
	if err != nil {
		// Handle the case where there are no more connections to accept.
		if err == unix.EAGAIN || err == unix.EWOULDBLOCK {
			return nil // This isn't necessarily an error, just no more connections to accept right now.
		}
		log.Logger.Error("accept error", zap.Error(err))
		return fmt.Errorf("accept error: %w", err)
	}

	// set the socket to non-blocking mode
	if err := unix.SetNonblock(connFd, true); err != nil {
		log.Logger.Error("set nonblock error", zap.Error(err))
		return fmt.Errorf("set nonblock error for fd %d: %w", connFd, err)
	}

	// register the new connection to epoll for read events
	if err := p.registerRead(connFd); err != nil {
		log.Logger.Error("register read error", zap.Error(err))
		return fmt.Errorf("register read error for fd %d: %w", connFd, err)
	}

	// print the ip address of the new connection
	var ip string
	switch addr := sa.(type) {
	case *unix.SockaddrInet4:
		ip = net.IPv4(addr.Addr[0], addr.Addr[1], addr.Addr[2], addr.Addr[3]).String()
	case *unix.SockaddrInet6:
		ip = net.IP(addr.Addr[:]).String() // Convert 16-byte slice to net.IP
	default:
		// Handle other address types or ignore
	}

	p.connPool[connFd] = &DefaultBufferedConn{
		fd: connFd,
		ip: ip,
	}

	// increase the number of fds
	p.incrFd()

	log.Logger.Debug("new connection", zap.Int("fd", connFd))

	return nil
}

func (p *Poll) incrFd() {
	atomic.AddInt64(&p.connCnt, 1)
}

func (p *Poll) decrFd() {
	atomic.AddInt64(&p.connCnt, -1)
}

func (p *Poll) handleWrite(conn BufferedConn) error {

	fd := conn.Fd()

	// Get the data to write
	data := conn.DataToWrite()
	n, err := p.writeRawToFd(fd, data)
	if err != nil {
		log.Logger.Error("write error", zap.Error(err))
		return fmt.Errorf("write error for fd %d: %w", fd, err)
	}

	// Advance the buffer to reflect the bytes written
	conn.Next(n)

	if conn.Len() == 0 {
		// All data was written. Deregister EPOLLOUT for this fd.
		if err := p.deregisterWrite(fd); err != nil {
			log.Logger.Error("failed to deregister write", zap.Error(err))
			return fmt.Errorf("failed to deregister write for fd %d: %w", fd, err)
		}
	}

	return nil
}

// writeRawToFd writes data to the socket
func (p *Poll) writeRawToFd(fd int, data []byte) (n int, err error) {
	n, err = unix.Write(fd, data)
	if err != nil {
		log.Logger.Error("write error", zap.Error(err))
		return n, err
	}
	return n, nil
}
