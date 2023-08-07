package main

type Poll struct {
	done chan struct{}
	*Registry
	epollFd  int   // epoll
	listenFD int   // listener fd
	connCnt  int64 // current fd size
	maxFD    int64 // max fd size,
	rHandler ReaderHandler

	//  "eventfd trick" to wake up a blocking system
	// used to send signal to epoll, trigger some event
	efd int

	connPool map[int]BufferedConn
}

func (p *Poll) SetHandler(handler ReaderHandler) {
	p.rHandler = handler
}
