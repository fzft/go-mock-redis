package main

import (
	"context"
	"golang.org/x/sys/unix"
	"sync"
)

// Registry is a wrapper around epoll. It keeps track of the connection fds that are registered to epoll.
type Registry struct {
	epollFd  int
	epollSet map[int]int
}

func NewRegistry(epollFd int) *Registry {
	return &Registry{
		epollFd:  epollFd,
		epollSet: make(map[int]int),
	}
}

// registerRead registers fd to epoll for read events.
func (r *Registry) registerRead(fd int) (err error) {
	_, ok := r.epollSet[fd]

	if ok {
		err = r.ModRead(fd)
	} else {
		err = r.AddRead(fd)
	}

	if err != nil {
		return err
	}

	r.epollSet[fd] = readEvents
	return
}

// registerWrite registers fd to epoll for write events.
func (r *Registry) registerWrite(fd int) (err error) {
	_, ok := r.epollSet[fd]

	if ok {
		err = r.ModWrite(fd)
	} else {
		err = r.AddWrite(fd)
	}

	if err != nil {
		return err
	}

	r.epollSet[fd] = writeEvents
	return
}

// deregisterWrite modifies fd in epoll to stop monitoring write events. turn to read events.
func (p *Poll) deregisterWrite(fd int) error {
	// Assuming you want to keep monitoring the fd for read events.
	// If not, adjust the events as necessary.
	return p.registerRead(fd)
}

// unregister removes fd from epoll.
func (r *Registry) unregister(fd int) (err error) {
	_, ok := r.epollSet[fd]

	if !ok {
		return nil
	}

	err = unix.EpollCtl(r.epollFd, unix.EPOLL_CTL_DEL, fd, nil)
	if err != nil {
		return err
	}

	delete(r.epollSet, fd)
	return
}

// RegisterReadWriter registers fd to epoll for read and write events.
func (r *Registry) RegisterReadWriter(fd int) (err error) {
	_, ok := r.epollSet[fd]

	if ok {
		err = r.ModReadWrite(fd)
	} else {
		err = r.AddReadWrite(fd)
	}

	if err != nil {
		return err
	}

	r.epollSet[fd] = readWriteEvents
	return
}

type Poll struct {
	ctx    context.Context
	doneCh chan struct{}
	*Registry
	epollFd        int   // epoll
	lnFd           int   // listener fd
	connCnt        int64 // current fd size
	maxConnections int64 // max fd size,
	rHandler       ReaderHandler

	mu       sync.Mutex
	connPool map[int]BufferedConn
}

func (p *Poll) Handler(handler ReaderHandler) {
	p.rHandler = handler
}
