package node

import (
	"net"
	"os"
)

type Reactor struct {
	listener net.Listener
	poll     *Poll
	done     chan struct{}
	sig      chan os.Signal
}

func NewReactor(listener net.Listener, sig chan os.Signal) (*Reactor, error) {
	r := &Reactor{
		listener: listener,
		done:     make(chan struct{}),
		sig:      sig,
	}

	f, err := listener.(*net.TCPListener).File()
	if err != nil {
		return nil, err
	}

	fd := int(f.Fd())
	p, err := NewPoll(r.done, MaxFD, fd)
	if err != nil {
		return nil, err
	}

	r.poll = p
	return r, nil
}

func (r *Reactor) Run() {
	go r.poll.poll()

	for {
		select {
		case <-r.done:
			return
		case <-r.sig:
			r.poll.sendSignal(SignalStop)
			<-r.done
			return
		}
	}
}

func (r *Reactor) SetHandler(handler ReaderHandler) {
	r.poll.SetHandler(handler)
}
