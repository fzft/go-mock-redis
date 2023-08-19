package node

import (
	"fmt"
	"github.com/fzft/go-mock-redis/commands"
	"github.com/fzft/go-mock-redis/db"
)

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
	ClientProtoTypeInline ClientProtoType = iota
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
	ClientPendingWrite     ClientFlags = 1 << (21 + iota) // Client has output to send but a write handler is yet not installed.
	ClientReplyOff                                        // Don't send replies to client.
	ClientReplySkipNext                                   // Set CLIENT_REPLY_SKIP_NEXT to skip next reply
	ClientReplySkip                                       // Don't send just this reply.
	ClientLuaDebug                                        // Run EVAL in debug mode.
	ClientLuaDebugSync                                    // Run EVAL in debug mode but don't start the script if the debugger is not attached.
	ClientModule                                          // Non connected client used by module API clients.
	ClientProtected                                       // Client should not be freed for now.
	ClientExecutingCommand                                /**Indicates that the client is currently in the process of handling
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
	id         uint64                   // client increment unique id
	flags      ClientFlags              // client type flags
	connection Conn                     // socket file descriptor
	resp       int                      // resp protocol version. Can be 2 or 3
	db         *db.RedisDb              // pointer to currently SELECTed DB
	queryBuf   []byte                   // buffer for client query
	queryPos   int                      // current position in query buffer
	argc       int                      // number of arguments in query buffer
	argv       [][]byte                 // arguments vector
	relies     *db.List[commands.Reply] // list of replies to send to client
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
		argv:       make([][]byte, 0),
		relies:     db.NewList[commands.Reply](),
	}
}

func (c *Client) GetID() uint64 {
	return c.id
}

/* -----------------------------------------------------------------------------
 * Higher level functions to queue data on the client output buffer.
 * The following functions are the ones that commands implementations will call.
 * -------------------------------------------------------------------------- */

// AddReply add the object 'obj' string representation to the client output buffer.
func (c *Client) AddReply(reply commands.Reply) {
	if !c.prepareClientToWrite() {
		return
	}
	if reply.EncodedObject() {
		c.addReplyToBufferOrList(reply.Marshal())
	} else if reply.Encoding == db.EncodingInt {
		c.addReplyToBufferOrList([]byte(fmt.Sprintf("%d", reply.Value.(int64))))
	} else {
		panic("Wrong reply encoding in AddReply() ")
	}

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

func (c *Client) addReplyToBufferOrList(data []byte) {

}
