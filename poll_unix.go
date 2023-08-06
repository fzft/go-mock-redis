//go:build linux
// +build linux

package main

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"
	"net"
	"os"
	"sync/atomic"
)

// https://copyconstruct.medium.com/the-method-to-epolls-madness-d9d2d6378642

const (
	readEvents      = unix.EPOLLPRI | unix.EPOLLIN
	writeEvents     = unix.EPOLLOUT
	readWriteEvents = readEvents | writeEvents
)

func NewPoll(ctx context.Context, doneCh chan struct{}, size int64, lnFd int) (*Poll, error) {
	// Create a new epoll instance
	epfd, err := unix.EpollCreate1(unix.EPOLL_CLOEXEC)
	if err != nil {
		Logger.Info("Failed to create epoll: %v", zap.Error(err))
		return nil, err
	}

	r := NewRegistry(epfd)

	// Register the listener to epoll for read events
	if err := r.AddRead(lnFd); err != nil {
		Logger.Error("Failed to add listener to epoll: %v", zap.Error(err))
		return nil, err
	}

	poll := &Poll{
		Registry:       r,
		epollFd:        epfd,
		lnFd:           lnFd,
		maxConnections: size,
		connPool:       make(map[int]BufferedConn),
		ctx:            ctx,
		doneCh:         doneCh,
	}

	return poll, nil
}

// CloseGracefully order: listener fd, all connections, epoll fd
func (p *Poll) CloseGracefully() error {

	// close the listener fd
	if err := p.Delete(p.lnFd); err != nil {
		Logger.Info("Failed to delete listener from epoll: %v", zap.Error(err))
	}

	if err := CloseFd(p.lnFd); err != nil {
		Logger.Info("Failed to close listener: %v", zap.Error(err))
	}

	// close all connections
	if err := p.ClosAndClearAllFDs(); err != nil {
		Logger.Info("Failed to close connections: %v", zap.Error(err))
	}

	// close the epoll fd
	if err := CloseFd(p.epollFd); err != nil {
		Logger.Info("Failed to close epoll: %v", zap.Error(err))
	}

	return nil
}

func (p *Poll) poll() {
	events := make([]unix.EpollEvent, p.maxConnections)
	msec := -1

	defer close(p.doneCh)

	// handle cleanup if necessary
	defer p.CloseGracefully()

	for {
		select {
		case <-p.ctx.Done():
			Logger.Info("Received stop signal. Exiting event loop.")
			return
		default:
			// EpollWait blocks until there is an event to report
			// n: number of events returned
			// if n ==0 , it means that the call timed out and no events were available
			// if n < 0, it means that an error occurred
			// level triggered, poll mode
			n, err := unix.EpollWait(p.epollFd, events, msec)
			if n == 0 || (n < 0 &&
				err == unix.EINTR) {
				Logger.Warn("epoll wait timeout")
				continue
			} else if err != nil {
				Logger.Error("epoll wait error: %v", zap.Error(err))
				return
			}

			for i := 0; i < n; i++ {
				ev := &events[i]
				Logger.Info("epoll event", zap.Int("fd", int(ev.Fd)), zap.Uint32("events", ev.Events))
				err := p.processEvent(int(ev.Fd), ev)
				if err != nil {
					Logger.Error("Failed to process event: %v", zap.Error(err))
					return
				}
			}
		}
	}
}

func (p *Poll) processEvent(fd int, ev *unix.EpollEvent) error {
	if ev.Events&unix.EPOLLERR != 0 || ev.Events&unix.EPOLLHUP != 0 {
		Logger.Info("epoll error event for fd %d", zap.Int("fd", fd))

		p.decrFd()

		// remove the fd from epoll set
		return p.unregister(fd)
	}

	// if the fd is the listener, it means that there is a new connection
	if ev.Fd == int32(p.lnFd) {
		return p.accept(fd)

	} else {
		// if the fd is not the listener, it means that there is data to read or write
		if ev.Events&unix.EPOLLIN != 0 {
			Logger.Debug("epoll read event for fd", zap.Int("fd", fd))
			conn, ok := p.connPool[fd]
			if !ok {
				Logger.Error("connection not found")
				return fmt.Errorf("connection not found for fd %d", fd)
			}

			return p.rHandler.Read(conn)
		}

		if ev.Events&unix.EPOLLOUT != 0 {
			Logger.Debug("epoll write event for fd %d", zap.Int("fd", fd))

			conn, ok := p.connPool[fd]
			if !ok {
				Logger.Error("connection not found")
				return fmt.Errorf("connection not found for fd %d", fd)
			}

			return p.handleWrite(conn)
		}
	}
	return nil
}

// accept a new connection
func (p *Poll) accept(fd int) error {
	connFd, sa, err := unix.Accept(fd)
	if err != nil {
		// Handle the case where there are no more connections to accept.
		if err == unix.EAGAIN || err == unix.EWOULDBLOCK {
			return nil // This isn't necessarily an error, just no more connections to accept right now.
		}
		Logger.Error("accept error: %v", zap.Error(err))
		return fmt.Errorf("accept error: %w", err)
	}

	// set the socket to non-blocking mode
	if err := unix.SetNonblock(connFd, true); err != nil {
		Logger.Error("set nonblock error: %v", zap.Error(err))
		return fmt.Errorf("set nonblock error for fd %d: %w", connFd, err)
	}

	// register the new connection to epoll for read events
	if err := p.registerRead(connFd); err != nil {
		Logger.Error("register read error: %v", zap.Error(err))
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

	p.mu.Lock()
	p.connPool[connFd] = &DefaultBufferedConn{
		fd: connFd,
		ip: ip,
	}
	p.mu.Unlock()

	// increase the number of fds
	p.incrFd()

	Logger.Debug("new connection", zap.Int("fd", connFd))

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
		Logger.Error("write error: %v", zap.Error(err))
		return fmt.Errorf("write error for fd %d: %w", fd, err)
	}

	// Advance the buffer to reflect the bytes written
	conn.Next(n)

	if conn.Len() == 0 {
		// All data was written. Deregister EPOLLOUT for this fd.
		if err := p.deregisterWrite(fd); err != nil {
			Logger.Error("failed to deregister write", zap.Error(err))
			return fmt.Errorf("failed to deregister write for fd %d: %w", fd, err)
		}
	}

	return nil
}

// writeRawToFd writes data to the socket
func (p *Poll) writeRawToFd(fd int, data []byte) (n int, err error) {
	n, err = unix.Write(fd, data)
	if err != nil {
		Logger.Error("write error", zap.Error(err))
		return n, err
	}
	return n, nil
}

// ClosAndClearAllFDs ...
func (r *Registry) ClosAndClearAllFDs() error {
	var errs MultiError

	// Close all registered file descriptors.
	for fd := range r.epollSet {
		if err := r.Delete(fd); err != nil {
			deleteErr := fmt.Errorf("delete fd: %d error: %v", fd, err)
			errs = append(errs, deleteErr)
			continue
		}
		if err := unix.Close(fd); err != nil {
			closeErr := fmt.Errorf("close fd: %d error: %v", fd, err)
			errs = append(errs, closeErr)
			continue
		}
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}

func (r *Registry) AddReadWrite(fd int) error {
	return os.NewSyscallError("epoll_ctl add",
		unix.EpollCtl(r.epollFd, unix.EPOLL_CTL_ADD, fd, &unix.EpollEvent{Fd: int32(fd), Events: readWriteEvents}))
}

func (r *Registry) AddRead(fd int) error {
	return os.NewSyscallError("epoll_ctl add",
		unix.EpollCtl(r.epollFd, unix.EPOLL_CTL_ADD, fd, &unix.EpollEvent{Fd: int32(fd), Events: readEvents}))
}

func (r *Registry) AddWrite(fd int) error {
	return os.NewSyscallError("epoll_ctl add",
		unix.EpollCtl(r.epollFd, unix.EPOLL_CTL_ADD, fd, &unix.EpollEvent{Fd: int32(fd), Events: writeEvents}))
}

func (r *Registry) ModRead(fd int) error {
	return os.NewSyscallError("epoll_ctl mod",
		unix.EpollCtl(r.epollFd, unix.EPOLL_CTL_MOD, fd, &unix.EpollEvent{Fd: int32(fd), Events: readEvents}))
}

func (r *Registry) ModWrite(fd int) error {
	return os.NewSyscallError("epoll_ctl mod",
		unix.EpollCtl(r.epollFd, unix.EPOLL_CTL_MOD, fd, &unix.EpollEvent{Fd: int32(fd), Events: writeEvents}))
}

func (r *Registry) ModReadWrite(fd int) error {
	return os.NewSyscallError("epoll_ctl mod",
		unix.EpollCtl(r.epollFd, unix.EPOLL_CTL_MOD, fd, &unix.EpollEvent{Fd: int32(fd), Events: readWriteEvents}))
}

func (r *Registry) Delete(fd int) error {
	return os.NewSyscallError("epoll_ctl del", unix.EpollCtl(r.epollFd, unix.EPOLL_CTL_DEL, fd, nil))
}
