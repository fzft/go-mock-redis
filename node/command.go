package node

import (
	"fmt"
	"github.com/fzft/go-mock-redis/db"
	"github.com/fzft/go-mock-redis/resp"
)

type CommandArgType uint8

const (
	ArgTypeString CommandArgType = iota
	ArgTypeInteger
	ArgTypeDouble
	ArgTypeKey
	ArgTypePattern
	ArgTypeUnixTime
	ArgTypePureToken
	ArgTypeOnEOF
	ArgTypeBlock
)

type CommandFlags uint64

const (
	CmdWrite CommandFlags = 1 << iota
	CmdReadOnly
	CmdDenyOOM
	CmdModule // Command exported by module.
	CmdAdmin
	CmdPubSub
	CmdNoScript
	_
	CmdBlocking // Has potential to block.
	CmdLoading
	CmdStale
	CmdSkipMonitor
	CmdSkipSlowLog
	CmdAsking
	CmdFast
	CmdNoAuth
	CmdMayReplicate
	CmdSentinel
	CmdOnlySentinel
	CmdNoMandatoryKeys
	CmdProtected
	CmdModuleGetKeys   // Use the modules getkeys interface.
	CmdModuleNoCluster // Deny on Redis Cluster.
	CmdNoAsyncLoading
	CmdNoMulti
	CmdMovableKeys // The legacy range spec doesn't cover all keys. Populated by populateCommandLegacyRangeSpec.
	_
	CmdAllowBusy
	CmdModuleGetChannels // Use the modules getchannels interface.
	CmdTouchesArbitraryKeys
)

type RedisCommandGroup uint8

const (
	RedisCommandGroupGeneric RedisCommandGroup = iota
	RedisCommandGroupString
	RedisCommandGroupList
	RedisCommandGroupSet
	RedisCommandGroupSortedSet
	RedisCommandGroupHash
	RedisCommandGroupPubSub
	RedisCommandGroupTransaction
	RedisCommandGroupConnection
	RedisCommandGroupServer
	RedisCommandGroupScripting
	RedisCommandGroupHyperLogLog
	RedisCommandGroupCluster
	RedisCommandGroupSentinel
	RedisCommandGroupGeo
	RedisCommandGroupStream
	RedisCommandGroupBitmap
	RedisCommandGroupModule
)

var (
	// Shared command responses

	SharedOk         = createRawStringObject(fmt.Sprintf("OK%s", resp.CRLF))
	SharedEmptyBulk  = createRawStringObject(fmt.Sprintf("%c0%s%s", resp.TypeBlob, resp.CRLF, resp.CRLF))
	SharedZCone      = createRawStringObject(fmt.Sprintf("%c0%s", resp.TypeInteger, resp.CRLF))
	SharedCone       = createRawStringObject(fmt.Sprintf("%c1%s", resp.TypeInteger, resp.CRLF))
	SharedEmptyArray = createRawStringObject(fmt.Sprintf("%c0%s", resp.TypeArray, resp.CRLF))
	SharedPong       = createRawStringObject(fmt.Sprintf("%cPONG%s", resp.TypeSimple, resp.CRLF))
	SharedQueued     = createRawStringObject(fmt.Sprintf("%cQUEUED%s", resp.TypeSimple, resp.CRLF))
	SharedEmptyScan  = createRawStringObject(fmt.Sprintf("%c2%s%c1%s0%s%c0%s", resp.TypeArray, resp.CRLF, resp.TypeBlob, resp.CRLF, resp.CRLF, resp.TypeArray, resp.CRLF))
	SharedSpace      = createRawStringObject(fmt.Sprintf(" "))
	SharedPlus       = createRawStringObject(fmt.Sprintf("%c", resp.TypeSimple))

	// Shared command error responses

	SharedWrongTypeErr   = createRawStringObject(fmt.Sprintf("%cWRONGTYPE Operation against a key holding the wrong kind of value%s", resp.TypeError, resp.CRLF))
	SharedErr            = createRawStringObject(fmt.Sprintf("%cERR%s", resp.TypeError, resp.CRLF))
	SharedNoKeyErr       = createRawStringObject(fmt.Sprintf("%cERR no such key%s", resp.TypeError, resp.CRLF))
	SharedSyntaxErr      = createRawStringObject(fmt.Sprintf("%cERR syntax error%s", resp.TypeError, resp.CRLF))
	SharedSomeObjErr     = createRawStringObject(fmt.Sprintf("%cERR source and destination objects are the same%s", resp.TypeError, resp.CRLF))
	SharedOutoffRangeErr = createRawStringObject(fmt.Sprintf("%cERR index out of range%s", resp.TypeError, resp.CRLF))
	SharedNoScriptErr    = createRawStringObject(fmt.Sprintf("%cNOSCRIPT No matching script. Please use EVAL.%s", resp.TypeError, resp.CRLF))
	SharedLoadingErr     = createRawStringObject(fmt.Sprintf("%cLOADING Redis is loading the dataset in memory%s", resp.TypeError, resp.CRLF))
	SharedSlowEvalErr    = createRawStringObject(fmt.Sprintf("%cBUSY Redis is busy running a script. You can only call SCRIPT KILL or SHUTDOWN NOSAVE.%s", resp.TypeError, resp.CRLF))
	SharedSlowScriptErr  = createRawStringObject(fmt.Sprintf("%cBUSY Redis is busy running a script. You can only call SCRIPT KILL or SHUTDOWN NOSAVE.%s", resp.TypeError, resp.CRLF))
	SharedNoAuthErr      = createRawStringObject(fmt.Sprintf("%cNOAUTH Authentication required.%s", resp.TypeError, resp.CRLF))
	ShardOOMErr          = createRawStringObject(fmt.Sprintf("%cOOM command not allowed when used memory > 'maxmemory'.%s", resp.TypeError, resp.CRLF))
	SharedExecAbortErr   = createRawStringObject(fmt.Sprintf("%cEXECABORT Transaction discarded because of previous errors.%s", resp.TypeError, resp.CRLF))
	SharedBusyKeyErr     = createRawStringObject(fmt.Sprintf("%cBUSYKEY Target key name already exists.%s", resp.TypeError, resp.CRLF))

	// The shared NULL depends on the protocol version, we just impl RESP3

	// SharedNull3 for RESP3
	SharedNull3 = createRawStringObject(fmt.Sprintf("%c%s", resp.TypeNull, resp.CRLF))

	// SharedNullArray3 for RESP3
	SharedNullArray3 = createRawStringObject(fmt.Sprintf("%c%s", resp.TypeNull, resp.CRLF))

	// SharedEmptySet3 for RESP3
	SharedEmptySet3 = createRawStringObject(fmt.Sprintf("%c0%s", resp.TypeSet, resp.CRLF))

	SharedMessageBulk      = createRawStringObject(fmt.Sprintf("%c7%smessage%s", resp.TypeBlob, resp.CRLF, resp.CRLF))
	SharedPmessageBulk     = createRawStringObject(fmt.Sprintf("%c8%spmessage%s", resp.TypeBlob, resp.CRLF, resp.CRLF))
	SharedSubscribeBulk    = createRawStringObject(fmt.Sprintf("%c9%ssubscribe%s", resp.TypeBlob, resp.CRLF, resp.CRLF))
	SharedUnsubscribeBulk  = createRawStringObject(fmt.Sprintf("%c11%sunsubscribe%s", resp.TypeBlob, resp.CRLF, resp.CRLF))
	SharedSSubscribeBulk   = createRawStringObject(fmt.Sprintf("%c10%sssubscribe%s", resp.TypeBlob, resp.CRLF, resp.CRLF))
	SharedSUnsubscribeBulk = createRawStringObject(fmt.Sprintf("%c12%ssunsubscribe%s", resp.TypeBlob, resp.CRLF, resp.CRLF))
	SharedSMessageBulk     = createRawStringObject(fmt.Sprintf("%c8%ssmessage%s", resp.TypeBlob, resp.CRLF, resp.CRLF))
	SharedPSubscribeBulk   = createRawStringObject(fmt.Sprintf("%c10%spsubscribe%s", resp.TypeBlob, resp.CRLF, resp.CRLF))
	SharedPUnsubscribeBulk = createRawStringObject(fmt.Sprintf("%c12%spunsubscribe%s", resp.TypeBlob, resp.CRLF, resp.CRLF))

	// Shared command names

	SharedDel       = createRawStringObject("DEL")
	SharedUnlink    = createRawStringObject("UNLINK")
	SharedRpop      = createRawStringObject("RPOP")
	SharedLPop      = createRawStringObject("LPOP")
	SharedLPush     = createRawStringObject("LPUSH")
	SharedRPopLPush = createRawStringObject("RPOPLPUSH")
	SharedLMove     = createRawStringObject("LMOVE")
	SharedBLMove    = createRawStringObject("BLMOVE")
	SharedZPopMin   = createRawStringObject("ZPOPMIN")
	SharedZPopMax   = createRawStringObject("ZPOPMAX")
	SharedMulti     = createRawStringObject("MULTI")
	SharedExec      = createRawStringObject("EXEC")
	SharedHSet      = createRawStringObject("HSET")
	SharedSRem      = createRawStringObject("SREM")
	SharedXGroup    = createRawStringObject("XGROUP")
	SharedXClaim    = createRawStringObject("XCLAIM")
	SharedScript    = createRawStringObject("SCRIPT")
	SharedReplConf  = createRawStringObject("REPLCONF")
	SharedPersist   = createRawStringObject("PERSIST")
	SharedSet       = createRawStringObject("SET")
	SharedEval      = createRawStringObject("EVAL")

	// Shared command argument

	SharedLeft            = createRawStringObject("left")
	SharedRight           = createRawStringObject("right")
	SharedPXAT            = createRawStringObject("PXAT")
	SharedTime            = createRawStringObject("TIME")
	SharedRetryCount      = createRawStringObject("RETRYCOUNT")
	SharedForce           = createRawStringObject("FORCE")
	SharedJustID          = createRawStringObject("JUSTID")
	SharedEntriesRead     = createRawStringObject("ENTRIESREAD")
	SharedLastID          = createRawStringObject("LASTID")
	SharedDefaultUsername = createRawStringObject("username")
	SharedPing            = createRawStringObject("ping")
	SharedSetId           = createRawStringObject("setid")
	SharedKeepTTL         = createRawStringObject("KEEPTTL")
	SharedABSTTL          = createRawStringObject("ABSTTL")
	SharedLoad            = createRawStringObject("LOAD")
	SharedCreateConsumer  = createRawStringObject("CREATECONSUMER")
	SharedGetACK          = createRawStringObject("GETACK")
	SharedSpecialAsterick = createRawStringObject("*")
	SharedSpecialEqual    = createRawStringObject("=")
	SharedRedacted        = createRawStringObject("(redacted)")
)

type MultiCmd struct {
	argv []*db.RedisObj
	argc int
	cmd  RedisCommand
}

type MultiState struct {
	commands []*MultiCmd // Array of commands in MULTI/EXEC context
	cmdFlags CommandFlags
}

type RedisCommandArgs struct {
	name    string
	numArgs int
}

type CommandHistory struct {
	since   string
	changes string
}

type RedisCommandProc func(c *Client) error

var ExecCommand RedisCommandProc = func(c *Client) error {
	return nil
}

type RedisCommand interface {
	Proc() RedisCommandProc //Command implementation
	DeclaredName() string
	Group() RedisCommandGroup
	History() []*CommandHistory
	SubCommands() []RedisCommand
	SubCommandsDict() *db.HashTable[string, RedisCommand]
	Args() []*RedisCommandArgs
	Arity() int
	ACLCategories() uint64
	Flags() CommandFlags

	// Runtime populated data

	Id() uint64 /* Command id,this is a progressive number starting from 0, and is used in order to check
	ACLs. A connection is able to execute a given command if the user associated to the connection*/

	MicroSeconds() int64
	GetCalls() int64
	SetCalls(int64)

	GetRejectedCalls() int64
	SetRejectedCalls(int64)

	GetFailedCalls() int64
	SetFailedCalls(int64)

	Parent() RedisCommand
	Fullname() string
}

type BaseCommand struct {
	declaredName    string
	proc            RedisCommandProc
	fullname        string
	group           RedisCommandGroup
	history         []*CommandHistory
	subCommands     []RedisCommand
	subCommandsDict *db.HashTable[string, RedisCommand]
	args            []*RedisCommandArgs
	arity           int
	aclCategories   uint64
	flags           uint64

	// Runtime populated data
	id            int
	microSeconds  int64
	calls         int64
	rejectedCalls int64
	failedCalls   int64
	parent        RedisCommand
}
