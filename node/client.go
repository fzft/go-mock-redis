package node

import (
	"bytes"
	"fmt"
	"github.com/fzft/go-mock-redis/db"
	"github.com/fzft/go-mock-redis/resp"
	"strings"
	"time"
)

const ConfigRunIdSize = 40

type BlockType uint8

const (
	BlockNone BlockType = iota
	BlockList
	BlockWait
	BlockModule
	BlockStream
	BlockZSet
	BlockPostPone
	BlockShutdown
	BlockNum
	BlockEnd
)

type ClientProtoType uint8

const (
	ClientProtoTypeUnknown ClientProtoType = iota
	ClientProtoTypeInline                  = iota
	ClientProtoTypeMultiBulk
)

type ClientFlags uint64

const (
	ClientSlave            ClientFlags                                    = 1 << iota // This client is a replica.
	ClientMaster                                                                      // This client is a master.
	ClientMonitor                                                                     // This client is a slave monitor, see MONITOR.
	ClientMulti                                                                       // This client is in a MULTI context.
	ClientBlocked                                                                     // The client is waiting in a blocking operation.
	ClientDirtyCas                                                                    // Watched keys modified. EXEC will fail.
	ClientCloseAfterReply                                                             //Close after writing entire reply.
	ClientUnblocked                                                                   // This client was unblocked and is stored in server.unblocked_clients
	ClientScript                                                                      //  This is a non connected client used by Lua
	ClientAsking                                                                      // Client issued the ASKING command
	ClientCloseASAP                                                                   // Close this client ASAP
	ClientUnixSocket                                                                  // Client connected via Unix domain socket
	ClientDirtyExec                                                                   // EXEC will fail for errors while queueing replies
	ClientMasterForceReply                                                            // Queue replies even if usually they are not queued (used for replication)
	ClientForceAOF                                                                    // Force AOF propagation of current cmd.
	ClientForceReplica                                                                // Force replica propagation of current cmd.
	ClientPrePSYNC                                                                    // Instance don't understand PSYNC.
	ClientReadOnly                                                                    // Cluster client is in read-only state
	ClientPubSub                                                                      // Client is in Pub/Sub mode
	ClientPreventAOFProp                                                              // Don't propagate to AOF, used by Lua for EVAL
	ClientPreventREPLProp                                                             // Don't propagate to replicas, used by Lua for EVAL
	ClientPreventProp      = ClientPreventAOFProp | ClientPreventREPLProp             // Don't propagate at all. Used by Lua for EVAL. Implies PREVENT_PROP_AOF and PREVENT_PROP_REPL
)

const (
	ClientPendingWrite  ClientFlags = 1 << (21 + iota) // Client has output to send but a write handler is yet not installed.
	ClientReplyOff                                     // Don't send replies to client.
	ClientReplySkipNext                                // Set CLIENT_REPLY_SKIP_NEXT to skip next reply
	ClientReplySkip                                    // Don't send just this reply.
	ClientLuaDebug                                     // Run EVAL in debug mode.
	ClientLuaDebugSync                                 // Run EVAL in debug mode but don't start the script if the debugger is not attached.
	ClientModule                                       // Non connected client used by module API clients.
	ClientProtected                                    // Client should not be freed for now.
	ClientExecutingCommand                             /**Indicates that the client is currently in the process of handling
		a command. usually this will be marked only during call()
	however, blocked clients might have this flag kept until they will try to reprocess the command **/
	ClientPendingCommand // Indicates the client has fully parsed command already for execution

	ClientTracking
	ClientTrackingBcast
	ClientTrackingOptIn
	ClientTrackingOptOut
	ClientTrackingCaching
	ClientTrackingNoLoop

	ClientInToTable
	ClientProtocolError
	ClientCloseAfterCommand
	ClientDenyBlocking
	ClientREPLRDBOnly
	ClientNoEvict
	ClientAllowOOM
	ClientNoTouch
	ClientPushing
	ClientModuleAuthHasResult
	ClientModulePreventAOFProp
	ClientModulePreventREPLProp
)

type ClientType uint8

const (
	ClientTypeNormal ClientType = iota
	ClientTypeSlave
	ClientTypePubSub
	ClientTypeMaster
	ClientTypeCount
)

type Client struct {
	id           uint64                 // client increment unique id
	flags        ClientFlags            // client type flags
	connection   Conn                   // socket file descriptor
	resp         int                    // resp protocol version. Can be 2 or 3
	db           *db.RedisDb            // pointer to currently SELECTed DB
	queryBuf     []byte                 // buffer for client query
	queryPos     int                    // current position in query buffer
	argc         int                    // number of arguments in query buffer
	argv         []*db.RedisObj         // arguments vector
	argvLen      int                    // Size of argv array (may be more than argc)
	argvLenSum   int                    // Sum of lengths of arguments
	replies      *db.List[*db.RedisObj] // list of reply to send to client
	cmd          RedisCommand           // command currently being processed
	lastCmd      RedisCommand           // command currently being processed
	realCmd      RedisCommand           // original command, if this is a replica
	reqType      ClientProtoType
	multiBulkLen int // number of multi bulk arguments left to read
	bulkLen      int // length of bulk argument in multi bulk request
	slot         int // The slot the client is executing against. Set to -1 if no slot is being used

	duration int64 // current command duration. Used for measuring latency of blocking commands

	readReplOff int64 // Read replication offset if this is a master.
	replOff     int64 // Applied replication offset if this is a master.
	replApplied int64 // Applied replication data count in querybuf, if this is a replica.
	replAckOff  int64 // Replication ack offset, if this is a replica.
	replAOFOff  int64 // AOF offset of the last fsync(), if this is my slave.
	replAckTime int64 // Replication ack time, if this is slave

	replId        [ConfigRunIdSize + 1]string // Master replication ID (if master)
	mState        *MultiState                 // MULTI/EXEC state
	authenticated bool                        // Needed when the default user requires auth.

}

func NewClient(id uint64, flags ClientFlags, connection Conn, resp int, rdb *db.RedisDb) *Client {
	return &Client{
		id:         id,
		flags:      flags,
		connection: connection,
		resp:       resp,
		db:         rdb,
		queryBuf:   make([]byte, 0),
		queryPos:   0,
		argc:       0,
		argv:       make([]*db.RedisObj, 0),
		replies:    db.NewList[*db.RedisObj](),
	}
}

/* Call is the core of redis execution of a command
* the following flags can be passed:
* CmdCallNone        No flags.
* CmdCallPROPAGATEAOF   Append command to AOF if it modified the dataset
*                          or if the client flags are forcing propagation.
* CMD_CALL_PROPAGATE_REPL  Send command to slaves if it modified the dataset
*                          or if the client flags are forcing propagation.
* CMD_CALL_PROPAGATE   Alias for PROPAGATE_AOF|PROPAGATE_REPL.
* CMD_CALL_FULL        Alias for SLOWLOG|STATS|PROPAGATE.
*
* The exact propagation behavior depends on the client flags.
* Specifically:
 */
func (c *Client) Call(flags int) {
	c.cmd.Proc()
}

// rejectCommand used when a command that is ready for execution needs to be rejected
func (c *Client) rejectCommand(reply *db.RedisObj) {
	c.duration = 0
	if c.cmd != nil {
		c.cmd.SetRejectedCalls(c.cmd.GetRejectedCalls() + 1)
		if c.cmd != nil && c.cmd.Fullname() == "ExecCommand" {
			c.execCommandAbort(reply.Value.(string))
		} else {
			c.addReplyErrorObject(reply)
		}
	}
}

// rejectCommandStr used when a command that is ready for execution needs to be rejected
func (c *Client) rejectCommandStr(reply string) {

	c.duration = 0
	if c.cmd != nil {
		c.cmd.SetRejectedCalls(c.cmd.GetRejectedCalls() + 1)
		if c.cmd != nil && c.cmd.Fullname() == "ExecCommand" {
			c.execCommandAbort(reply)
		} else {
			c.addReplyErrorStr(reply)
		}
	}
}

func (c *Client) rejectCommandFormat(fmtstr string, a ...string) {
	s := fmt.Sprintf(fmtstr, a)
	s = mapChars(s, "\r\n", "  ")
	c.rejectCommandStr(s)
}

func (c *Client) GetID() uint64 {
	return c.id
}

/* -----------------------------------------------------------------------------
 * Higher level functions to queue data on the client output buffer.
 * The following functions are the ones that commands implementations will call.
 * -------------------------------------------------------------------------- */

// AddReply add the object 'obj' string representation to the client output buffer.
func (c *Client) AddReply(reply *db.RedisObj) {
	if !c.prepareClientToWrite() {
		return
	}
	if reply.EncodingObject() {
		c.connection.Write([]byte(reply.Value.(string)))
	} else if reply.Encoding == db.EncodingInt {
		c.connection.Write([]byte(fmt.Sprintf("%d", reply.Value.(int64))))
	} else {
		panic("Wrong reply encoding in AddReply() ")
	}

}

// addReplyProto this low level function just adds whatever protocol you send it to the
// client
func (c *Client) addReplyProto(proto []byte) {
	if !c.prepareClientToWrite() {
		return
	}
	c.connection.Write(proto)
}

// AddReplyBulk ...
func (c *Client) AddReplyBulk(obj *db.RedisObj) {
	c.addReplyBulkLen(obj)
	c.AddReply(obj)
	c.addReplyProto([]byte(resp.CRLF))
}

// AddReplyError ...
func (c *Client) AddReplyError(err string) {
	c.addReplyErrorLength(err)
	c.afterErrorReply(err)
}

// addReplyErrorLength
// low level function called by the AddReplyError...() functions
// It emits the protocol for a redis error, in the form:
// -ERRORCODE Error Message\r\n
func (c *Client) addReplyErrorLength(err string) {
	if len(err) > 0 && err[0] == '-' {
		c.addReplyProto([]byte("-ERR"))
	}
	c.addReplyProto([]byte(err))
	c.addReplyProto([]byte(resp.CRLF))
}

// AddReplyErrorExpireTime ...
func (c *Client) AddReplyErrorExpireTime() {
	c.addReplyErrorFormat(fmt.Sprintf("invalid expire time in '%s' command", c.cmd.Fullname()))
}

// addReplyErrorFormat
func (c *Client) addReplyErrorFormat(err string) {
	c.addReplyErrorLength(err)
	c.afterErrorReply(err)
}

// addReplyErrorObject
func (c *Client) addReplyErrorObject(err *db.RedisObj) {
	c.AddReply(err)
	c.afterErrorReply(err.Value.(string))
}

// addReplyErrorStr
func (c *Client) addReplyErrorStr(err string) {
	c.addReplyErrorLength(err)
	c.afterErrorReply(err)
}

// afterErrorReply ...
// TODO:
func (c *Client) afterErrorReply(err string) {
}

// addReplyBulkLen
/* Create the length prefix of a bulk reply, example: $2234 */
func (c *Client) addReplyBulkLen(obj *db.RedisObj) {
	l := stringObjectLen(obj)
	c.addReplyLongLongWithPrefix('$', int64(l))
}

// addReplyLongLongWithPrefix
func (c *Client) addReplyLongLongWithPrefix(prefix byte, ll int64) {
	buf := ll2String(prefix, ll)
	c.addReplyProto(buf)
}

// prepareClientToWrite
// this function is called every time we are going to transmit new data to the client.
// the behavior is the following:
// 1) If the client should recv new data the function return true
// make sure to install the write handler in our event loop so that when the socket is writable new data gets written.
// 2) If the client should not recv new data the function return false
func (c *Client) prepareClientToWrite() bool {

	// If it's the lua client we always return true.
	if c.flags&ClientScript != 0 {
		return true
	}

	// If CLIENT_CLOSE_ASAP flag is set, we need not write anything
	if c.flags&ClientCloseASAP != 0 {
		return false
	}

	/* CLIENT REPLY OFF / SKIP handling: don't send replies.
	 * CLIENT_PUSHING handling: disables the reply silencing flags. */
	if c.flags&(ClientReplyOff|ClientReplySkip) != 0 && c.flags&ClientPushing == 0 {
		return false
	}

	/* Masters don't receive replies, unless CLIENT_MASTER_FORCE_REPLY flag
	 * is set. */
	if c.flags&ClientMaster != 0 && c.flags&ClientMasterForceReply == 0 {
		return false
	}

	if c.connection == nil {
		return false
	}

	return true
}

/* processInputBuffer This function is called every time, in the client structure 'c', there is
* more query buffer to process, because we read more data from the socket
* or because a client was blocked and later reactivated, so there could be
* pending query buffer, already representing a full command, to process.
* return false in case the client was freed during the processing */
func (c *Client) processInputBuffer() bool {

	for c.queryPos < len(c.queryBuf) {
		if c.flags&ClientBlocked != 0 ||
			c.flags&ClientPendingCommand != 0 {
			break
		}

		/* CLIENT_CLOSE_AFTER_REPLY closes the connection once the reply is
		 * written to the client. Make sure to not let the reply grow after
		 * this flag has been set (i.e. don't process more commands).
		 *
		 * The same applies for clients we want to terminate ASAP. */
		if c.flags&(ClientCloseAfterReply|ClientCloseASAP) != 0 {
			break
		}

		// Determine request type if unknown.
		if c.reqType == ClientProtoTypeUnknown {
			if c.queryBuf[c.queryPos] == '*' {
				c.reqType = ClientProtoTypeMultiBulk
			} else {
				c.reqType = ClientProtoTypeInline
			}
		} else {
			panic("Unknown client reqtype")
		}

		if c.argc == 0 {
			c.resetClient()
		} else {
			// we are finally ready to execute the command
			if err := c.processCommandAndResetClient(); err != nil {
				/* If the client is no longer valid, we avoid exiting this
				 * loop and trimming the client buffer later. So we return
				 * ASAP in that case. */
				return false
			}
		}
	}

	if c.flags&ClientMaster != 0 {
		/* If the client is a master, trim the querybuf to repl_applied,
		 * since master client is very special, its querybuf not only
		 * used to parse command, but also proxy to sub-replicas.
		 *
		 * Here are some scenarios we cannot trim to qb_pos:
		 * 1. we don't receive complete command from master
		 * 2. master client blocked cause of client pause
		 * 3. io threads operate read, master client flagged with CLIENT_PENDING_COMMAND
		 *
		 * In these scenarios, qb_pos points to the part of the current command
		 * or the beginning of next command, and the current command is not applied yet,
		 * so the repl_applied is not equal to qb_pos. */
		if c.replApplied > 0 {
			c.queryBuf = c.queryBuf[c.replApplied:]
			c.queryPos = int(c.replApplied)
			c.replApplied = 0
		}
	} else if c.queryPos > 0 {
		// Trim to pos.
		c.queryBuf = c.queryBuf[c.queryPos:]
		c.queryPos = 0
	}

	return true
}

// processInlineBuffer for the inline protocol instead of RESP
// this function consume the client query buffer and creates a command ready
// to be executed. or returns C_ERR if the client query buffer is not
func (c *Client) processInlineBuffer() bool {

	var linefeedChars = 1

	// Search for end of line
	p := bytes.IndexByte(c.queryBuf[c.queryPos:], '\n')

	// Nothing to do without a \r\n
	if p == -1 {
		if len(c.queryBuf)-c.queryPos >= ProtoInlineMaxSize {
			c.AddReplyError("Protocol error: too big inline request")
			c.queryBuf = make([]byte, 0)
			c.queryPos = 0
		}
	}

	// Handle the \r\n case.
	if p != 0 && c.queryBuf[p-1] == '\r' {
		p--
		linefeedChars++
	}

	queryLen := p - c.queryPos
	aux := string(c.queryBuf[c.queryPos : c.queryPos+queryLen])

	// Splitting the string into an array (slice in Go) of strings
	argv := strings.Fields(aux)

	// Check if argv could not be created, perhaps due to unbalanced quotes
	// (In the real world, you'd actually try to detect this more rigorously)
	if argv == nil {
		// Do error handling, e.g., send a reply or set an error flag
		c.AddReplyError("Protocol error: unbalanced quotes in request")
		return false
	}

	if queryLen == 0 && c.flags == ClientSlave {
		c.replAckTime = time.Now().Unix()
	}

	// TODO: ClientMaster

	c.queryPos += queryLen + linefeedChars
	argvLen := len(argv)

	if argvLen > 0 {
		c.argvLen = argvLen
		c.argv = make([]*db.RedisObj, c.argvLen)
		c.argvLenSum = 0
	}

	// create redis object for each argument
	for _, arg := range argv {
		newObj := createObject(db.StringType, arg) // Assuming CreateObject returns a new Redis object
		c.argv = append(c.argv, newObj)
		c.argc++
		c.argvLenSum += len(arg)
	}

	return true
}

// resetClient prepare the client to process the next command
func (c *Client) resetClient() {

	var prevCmd RedisCommand

	if c.cmd != nil {
		prevCmd = c.cmd
	}

	c.reqType = ClientProtoTypeUnknown
	c.multiBulkLen = 0
	c.bulkLen = -1
	c.slot = -1
	c.flags &= ^ClientExecutingCommand

	if c.flags&ClientMulti == 0 && prevCmd.Fullname() != "asking" {
		c.flags &= ^ClientAsking
	}

	if c.flags&ClientMulti == 0 && prevCmd.Fullname() != "client" {
		c.flags &= ^ClientTrackingCaching
	}

	c.flags &= ^ClientReplySkip
	if c.flags&ClientReplySkipNext != 0 {
		c.flags |= ClientReplySkip
		c.flags &= ^ClientReplySkipNext
	}
}

func (c *Client) processCommandAndResetClient() error {
	if c.processCommand() {

	}

	return nil
}

// processCommand if this function gets called we already read a whole
// command , arguments are in c.argv/argc fields
func (c *Client) processCommand() bool {
	// TODO:script is timeout
	var clientReprocessingCommand bool
	if c.cmd != nil {
		clientReprocessingCommand = true
	}

	// Handle possible security attacks.
	if strings.EqualFold(c.argv[0].Value.(string), "host:") || strings.EqualFold(c.argv[0].Value.(string), "post") {
		return false
	}

	// Now lookup the command and check ASAP about trivial error conditions
	// such as wrong arity, bad command name and so forth.
	// in case of error the function returns early
	if !clientReprocessingCommand {
		c.cmd = c.lookupCommand(c.argv, c.argc)
		c.lastCmd = c.cmd
		c.realCmd = c.cmd
		if err, ok := c.commandCheckExistence(); !ok {
			c.rejectCommandStr(err)
			return true
		}

		if err, ok := c.commandCheckArity(); !ok {
			c.rejectCommandStr(err)
			return true
		}
		//check if the command is marked as protected and the relevant configuration allows it
		if c.cmd.Flags()&CmdProtected != 0 {
			if c.cmd.Fullname() == "debugCommand" || c.cmd.Fullname() == "module" {
				var cmdEnable, cmdName string
				if c.cmd.Fullname() == "debugCommand" {
					cmdName = "DEBUG"
					cmdEnable = "enable-debug-command"
				} else {
					cmdName = "MODULE"
					cmdEnable = "enable-module-command"
				}
				c.rejectCommandFormat("%s command not allowed. If the %s option is set to \"local\" you can run it from a local connection, otherwise you need to set this option in the configuration file, and then restart the server.", cmdName, cmdEnable)
				return true
			}
		}
	}

	//cmdFlags := c.cmd.Flags()
	//isReadCmd := cmdFlags&CmdReadOnly != 0 || (c.cmd.Fullname() == "execCommand" && c.mState.cmdFlags&CmdReadOnly != 0)
	//isWriteCmd := cmdFlags&CmdWrite != 0 || (c.cmd.Fullname() == "execCommand" && c.mState.cmdFlags&CmdWrite != 0)
	//isDenyOOMCmd := cmdFlags&CmdDenyOOM != 0 || (c.cmd.Fullname() == "execCommand" && c.mState.cmdFlags&CmdDenyOOM != 0)
	//isDenyStableCmd := cmdFlags&CmdStale != 0 || (c.cmd.Fullname() == "execCommand" && c.mState.cmdFlags&CmdStale != 0)
	//isDenyLoadingCmd := cmdFlags&CmdLoading != 0 || (c.cmd.Fullname() == "execCommand" && c.mState.cmdFlags&CmdLoading != 0)
	//isMayReplCmd := cmdFlags&(CmdWrite|CmdMayReplicate) != 0 || (c.cmd.Fullname() == "execCommand" && c.mState.cmdFlags&(CmdWrite|CmdMayReplicate) != 0)
	//isDenyAsyncLoadingCmd := cmdFlags&CmdNoAsyncLoading != 0 || (c.cmd.Fullname() == "execCommand" && c.mState.cmdFlags&CmdNoAsyncLoading != 0)
	//isObeyClient := c.mustObeyClient()

	if c.authRequired() {
		/* AUTH and HELLO and no auth commands are valid even in
		 * non-authenticated state. */
		if c.cmd.Flags()&CmdNoAuth == 0 {
			c.rejectCommand(SharedNoAuthErr)
			return true
		}
	}

	if c.flags&ClientMulti != 0 && c.cmd.Flags() == CmdNoMulti {
		c.rejectCommandFormat("Command not allowed inside a transaction")
		return true
	}

	// check if the user can run this command according to the current Acls

	return false

}

// AuthRequired check the user is authenticated. this check is skipped in case
// the default user is flagged as "nopass" and the client is authenticated
func (c *Client) authRequired() bool {
	return (!(defaultUser.flags&UserFlagNoPass == 0 || defaultUser.flags&UserFlagDisabled != 0)) && !c.authenticated
}

func (c *Client) mustObeyClient() bool {
	return c.flags&ClientMaster == 0
}

// commandCheckExistence
func (c *Client) commandCheckExistence() (string, bool) {
	if c.cmd != nil {
		return "", true
	}

	var err string
	if c.isContainerCommandBySds(c.argv[0].Value.(string)) {
		// if we can't find the command, but argv[0] by it self is a command
		// it means we're dealing with a invalid subcommand. Print Help
		cmdStr := strings.ToUpper(c.argv[0].Value.(string))
		err = fmt.Sprintf("Unknown subcommand or wrong number of arguments for '%s'. Try CLIENT HELP.", cmdStr)
	} else {
		args := ""
		limit := 128
		for _, arg := range c.argv[1:] {
			remaining := limit - len(args)
			if remaining <= 0 {
				break
			}
			fragment := fmt.Sprintf("'%.*s' ", remaining, arg)
			args += fragment
		}
		err = fmt.Sprintf("unknown command '%.128s', with args beginning with: %s", c.argv[0], args)
	}

	err = mapChars(err, "\r\n", "  ")
	return err, false
}

// commandCheckArity Check if c.args is valid for c.cmd, fill 'error' details in case it is not.
func (c *Client) commandCheckArity() (string, bool) {
	if (c.cmd == nil && c.cmd.Arity() > 0 && c.cmd.Arity() != c.argc) || (c.argc < -c.cmd.Arity()) {
		return fmt.Sprintf("wrong number of arguments for '%s' command", c.cmd.Fullname()), false
	}
	return "", true
}

func (c *Client) isContainerCommandBySds(s string) bool {
	baseCmd, exist := server.commands.Get(s)
	if exist && baseCmd.SubCommands() != nil {
		return true
	}
	return false
}

// lookupCommand
func (c *Client) lookupCommand(argv []*db.RedisObj, argc int) RedisCommand {
	return c.lookupCommandLogic(server.commands, argv, argc, false)

}

// lookupCommandLogic lookup a command by argv and argc
// if strict is not 0 we expect argc to be exact `strict` should be used every time we want to look up a command name
// rather than to find the command a user requested to execute
func (c *Client) lookupCommandLogic(commands *db.HashTable[string, RedisCommand], argv []*db.RedisObj, argc int, strict bool) RedisCommand {

	var hasSubCommands bool
	baseCmd, exist := commands.Get(argv[0].Value.(string))
	if exist && baseCmd.SubCommands() != nil {
		hasSubCommands = true
	}

	if argc == 1 || !hasSubCommands {
		if strict && argc != 1 {
			return nil
		}
		return baseCmd
	} else {

		if strict && argc != 2 {
			return nil
		}
		return c.lookupSubCommand(baseCmd, argv[1].Value.(string))
	}
}

// lookupSubCommand
func (c *Client) lookupSubCommand(baseCmd RedisCommand, sub string) RedisCommand {
	// TODO: lookup sub command
	return nil
}
