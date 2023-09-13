package hredis

type RedisConnectionType uint8

const (
	RedisConnTCP RedisConnectionType = iota
	RedisConnUnix
	RedisConnUserFd
)

type RedisOptType byte

const (
	RedisOptNonBlock          RedisOptType = 0x01
	RedisOptReuseAddr         RedisOptType = 0x02
	RedisOptNoAutoFree        RedisOptType = 0x04
	RedisOptNoPushAutoFree    RedisOptType = 0x08
	RedisOptNoAutoFreeReplies RedisOptType = 0x10
	RedisOptPreferIPv4        RedisOptType = 0x20
	RedisOptPreferIPv6        RedisOptType = 0x40
	RedisOptPreferIpUnspec    RedisOptType = RedisOptPreferIPv4 | RedisOptPreferIPv6
)

type Endpoint struct {
	sourceAddr string
	ip         string
	port       int

	redisFd int

	// use this field for unix domain sockets
	unixSocket string
}

type RedisOpts struct {
	Endpoint

	/*
	 * the type of connection to use. This also indicates which
	 * `endpoint` member field to use
	 */
	tp int

	// bit field of RedisOptType
	options RedisOptType
}

func NewRedisOpts() *RedisOpts {
	return &RedisOpts{}
}

type ConnectionType int

const (
	RedisBlock             ConnectionType = 0x01
	RedisConnected         ConnectionType = 0x02
	RedisDisconnecting     ConnectionType = 0x04
	RedisFreeing           ConnectionType = 0x08
	RedisInCallBack        ConnectionType = 0x10
	RedisSubscribed        ConnectionType = 0x20
	RedisMonitoring        ConnectionType = 0x40
	RedisReusedAddr        ConnectionType = 0x80
	RedisSupportPush       ConnectionType = 0x100
	RedisNoAutoFree        ConnectionType = 0x200
	RedisNoAutoFreeReplies ConnectionType = 0x400
	RedisPreferIPv4        ConnectionType = 0x800
	RedisPreferIPv6        ConnectionType = 0x1000
)

// RedisContext is context for
type RedisContext struct {
	redisFd int

	flags ConnectionType
}

func RedisConnect(ip string, port int) *RedisContext {
	opts := NewRedisOpts()
	opts.ip = ip
	opts.port = port
	return NewRedisContextWithOpts(opts)
}

func NewRedisContextWithOpts(opts *RedisOpts) *RedisContext {
	c := &RedisContext{}

	if opts.options&RedisOptNonBlock == 0 {
		c.flags |= RedisBlock
	}

	if opts.options&RedisOptReuseAddr != 0 {
		c.flags |= RedisReusedAddr
	}

	if opts.options&RedisOptNoAutoFree != 0 {
		c.flags |= RedisNoAutoFree
	}

	if opts.options&RedisOptNoAutoFreeReplies != 0 {
		c.flags |= RedisNoAutoFreeReplies
	}

	if opts.options&RedisOptPreferIPv4 != 0 {
		c.flags |= RedisPreferIPv4
	}

	if opts.options&RedisOptPreferIPv6 != 0 {
		c.flags |= RedisPreferIPv6
	}

	return c
}
