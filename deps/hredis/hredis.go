package hredis

import (
	"fmt"
	"strconv"
)

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
	RedisFd    int
	Err        RedisErrFlag
	ErrStr     string
	SocketAddr string
	oBuf       []byte // output buffer
	reader     *RedisReader

	Tcp struct {
		Host       string
		SourceAddr string
		Port       int
	}

	UnixSocket struct {
		Path string
	}

	Flags ConnectionType
}

func RedisConnect(ip string, port int) *RedisContext {
	opts := NewRedisOpts()
	opts.ip = ip
	opts.port = port
	return NewRedisContextWithOpts(opts)
}

func RedisConnectUnix(unixSocket string) *RedisContext {
	opts := NewRedisOpts()
	opts.unixSocket = unixSocket
	return NewRedisContextWithOpts(opts)
}

func NewRedisContextWithOpts(opts *RedisOpts) *RedisContext {
	c := &RedisContext{}

	if opts.options&RedisOptNonBlock == 0 {
		c.Flags |= RedisBlock
	}

	if opts.options&RedisOptReuseAddr != 0 {
		c.Flags |= RedisReusedAddr
	}

	if opts.options&RedisOptNoAutoFree != 0 {
		c.Flags |= RedisNoAutoFree
	}

	if opts.options&RedisOptNoAutoFreeReplies != 0 {
		c.Flags |= RedisNoAutoFreeReplies
	}

	if opts.options&RedisOptPreferIPv4 != 0 {
		c.Flags |= RedisPreferIPv4
	}

	if opts.options&RedisOptPreferIPv6 != 0 {
		c.Flags |= RedisPreferIPv6
	}

	return c
}

// RedisReply is a reply object returned by the RedisCommand
type RedisReply struct {
	Tp       RedisReplyType
	Integer  int64   // The integer when type is RedisReplyInteger
	Dval     float64 // The double when type is RedisReplyDouble
	Str      string  // The string when type is RedisReplyString,RedisReplyError,RedisReplyStatus
	Vtype    [4]byte // The type of the vector when type is RedisReplyVector
	Elements int     // number of elements, for RedisReplyArray
	Element  []*RedisReply
}

func (c *RedisContext) RedisCommand(format string, args ...interface{}) *RedisReply {
	return c.RedisVCommand(format, args)
}

func (c *RedisContext) RedisVCommand(format string, args ...interface{}) *RedisReply {
	if c.redisVAppendCommand(format, args...) != RedisOk {
		return nil
	}

	return c.redisBlockForReply()
}

func (c *RedisContext) redisVAppendCommand(format string, args ...interface{}) RedisStatus {
	cmd, err := redisFormatCommand(format, args...)
	if err != nil {
		return RedisErr
	}

	if ok := redisAppendCommand(c, cmd); ok != RedisOk {
		return RedisErr
	}

	return RedisOk
}

/* redisBlockForReply write a formatted command to the output buffer.
* If the context is blocking, immediately read the reply
* Write a formatted command to the output buffer. If the given context is
* blocking, immediately read the reply into the "reply" pointer. When the
* context is non-blocking, the "reply" pointer will not be used and the
* command is simply appended to the write buffer.
*
* Returns the reply when a reply was successfully retrieved. Returns NULL
* otherwise. When NULL is returned in a blocking context, the error field
* in the context will be set.*/
func (c *RedisContext) redisBlockForReply() *RedisReply {
	if c.Flags&RedisBlock != 0 {
		if reply, ok := c.redisGetReply(); ok == RedisOk {
			return reply
		}
	}
	return nil
}

func (c *RedisContext) redisGetReply() (*RedisReply, RedisStatus) {
	var (
		aux *RedisReply
		ok  RedisStatus
	)

	// try to read pending replies
	if ok = c.redisNextInBandReplyFromReader(aux); ok != RedisOk {
		return nil, RedisErr
	}

	return aux, RedisOk
}

// redisNextInBandReplyFromReader
/* Internal helper to get the next reply from our reader while handling
 * any PUSH messages we encounter along the way.  This is separate from
 * redisGetReplyFromReader so as to not change its behavior. */
func (c *RedisContext) redisNextInBandReplyFromReader(reply *RedisReply) RedisStatus {

	var ok RedisStatus

	for {
		if ok = c.redisGetReplyFromReader(reply); ok != RedisOk {
			return RedisErr
		}

		if !c.redisHandledPushReply(reply) {
			break
		}
	}
	return RedisOk
}

// redisGetReplyFromReader get a reply from the reader or set an error in the context
func (c *RedisContext) redisGetReplyFromReader(reply *RedisReply) RedisStatus {
	if ok := c.reader.GetReply(reply); ok != RedisOk {
		c.SetError(c.reader.err, c.reader.errStr)
		return RedisErr
	}
	return RedisOk
}

func (c *RedisContext) SetError(tp RedisErrFlag, err string) {
	c.Err = tp
	if err != "" {
		c.ErrStr = err
	} else {
		// TODO Only REDIS_ERR_IO may lack a description!
	}
}

// redisHandledPushReply internal helper that returns true if the reply is a RESP3 PUSH message
func (c *RedisContext) redisHandledPushReply(reply *RedisReply) bool {
	if reply != nil && reply.Tp == RedisReplyPush {
		return true
	}
	return false
}

// redisAppendCommand write a formatted command to the output buffer
func redisAppendCommand(c *RedisContext, cmd []string) RedisStatus {
	for _, str := range cmd {
		c.oBuf = append(c.oBuf, []byte(str)...)
	}
	return RedisOk
}

func redisFormatCommand(format string, args ...interface{}) ([]string, error) {

	var curArg string
	var argv []string

	argIndex := 0 // To track the current argument in args
	touched := false

	for i := 0; i < len(format); i++ {
		c := format[i]
		if c != '%' {
			if c == ' ' {
				if touched {
					argv = append(argv, curArg)
					curArg = ""
					touched = false
				}
			} else {
				curArg += string(c)
				touched = true
			}
		} else {
			i++
			if i >= len(format) {
				return nil, fmt.Errorf("Format string ended unexpectedly")
			}

			switch format[i] {
			case 's':
				if argIndex >= len(args) {
					return nil, fmt.Errorf("Not enough arguments")
				}
				str, ok := args[argIndex].(string)
				if !ok {
					return nil, fmt.Errorf("Expected a string argument")
				}
				curArg += str
				argIndex++

			case 'd':
				if argIndex >= len(args) {
					return nil, fmt.Errorf("Not enough arguments")
				}
				num, ok := args[argIndex].(int)
				if !ok {
					return nil, fmt.Errorf("Expected an integer argument")
				}
				curArg += strconv.Itoa(num)
				argIndex++

			default:
				return nil, fmt.Errorf("Unsupported format specifier: %c", format[i])
			}

			touched = true
		}
	}

	if touched {
		argv = append(argv, curArg)
	}

	return argv, nil
}
