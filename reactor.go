package main

import (
	"context"
	"go.uber.org/zap"
	"net"
	"os"
)

type Reactor struct {
	ln         net.Listener
	poll       *Poll
	cancelFunc context.CancelFunc
	doneCh     chan struct{}
	signal     chan os.Signal
}

func NewReactor(ln net.Listener, signal chan os.Signal) (*Reactor, error) {
	r := new(Reactor)
	ctx, cancel := context.WithCancel(context.Background())
	doneCh := make(chan struct{})

	f, err := ln.(*net.TCPListener).File()
	if err != nil {
		Logger.Info("Failed to get listener fd: %v", zap.Error(err))
		return nil, err
	}

	lnFd := int(f.Fd())
	poll, err := NewPoll(ctx, doneCh, MaxFD, lnFd)
	if err != nil {
		return nil, err
	}

	r.poll = poll
	r.ln = ln
	r.doneCh = doneCh
	r.signal = signal
	r.cancelFunc = cancel

	return r, nil
}

func (r *Reactor) Run() {
	go r.poll.poll()
	defer Logger.Info("reactor closed")

	for {
		select {
		case <-r.doneCh:
			return
		case <-r.signal:
			Logger.Info("signal received")
			r.cancelFunc()
			<-r.doneCh
			return
		}
	}
}

func (r *Reactor) RegisterRead(fd int) error {
	return r.poll.registerRead(fd)
}

func (r *Reactor) RegisterWrite(fd int) error {
	return r.poll.registerWrite(fd)
}

func (r *Reactor) Unregister(fd int) error {
	return r.poll.unregister(fd)
}

func (r *Reactor) Handler(handler ReaderHandler) {
	r.poll.Handler(handler)
}
