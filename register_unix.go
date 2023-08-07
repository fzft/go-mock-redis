//go:build linux
// +build linux

package main

import (
	"fmt"
	"golang.org/x/sys/unix"
	"os"
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
func (r *Registry) deregisterWrite(fd int) error {
	// Assuming you want to keep monitoring the fd for read events.
	// If not, adjust the events as necessary.
	return r.registerRead(fd)
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
