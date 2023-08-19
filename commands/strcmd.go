package commands

import (
	"github.com/fzft/go-mock-redis/db"
	"github.com/fzft/go-mock-redis/node"
)

type StrSetType int

const (
	ObjNoFlags    StrSetType = iota
	ObjSetNX                 = 1 << 0 // Set if key not exists.
	ObjSetXX                 = 1 << 1 // Set if key exists.
	ObjSetEX                 = 1 << 2 // Set if time in seconds is give.
	ObjSetPX                 = 1 << 3 // Set if time in milliseconds is given.
	ObjSetKeepTTL            = 1 << 4 // Keep the TTL if the key exists.
	ObjSetGet                = 1 << 5 // Set if want to get key before set.
	ObjEXAT                  = 1 << 6 // Set if time in seconds is given as timestamp.
	ObjPXAT                  = 1 << 7 // Set if time in milliseconds is given as timestamp.
	ObjPERSIST               = 1 << 8 // Set if want to remove expire.
)

// StrCmd handles string commands.
type StrCmd struct {
	c  *node.Client
	db *db.RedisDb
}

// NewStrCmd returns a new StrCmd.
func NewStrCmd(c *node.Client) *StrCmd {
	return &StrCmd{c: c}
}

/* setGenericCommand function implements the SET operation with different
 * options and variants. This function is called in order to implement the
 * following commands: SET, SETEX, PSETEX, SETNX, GETSET.
 *
 * 'flags' changes the behavior of the command (NX, XX or GET, see below).
 *
 * 'expire' represents an expire to set in form of a Redis object as passed
 * by the user. It is interpreted according to the specified 'unit'.
 *
 * 'ok_reply' and 'abort_reply' is what the function will reply to the client
 * if the operation is performed, or when it is not because of NX or
 * XX flags.
 *
 * If ok_reply is NULL "+OK" is used.
 * If abort_reply is NULL, "$-1" is used. */
func (cmd *StrCmd) setGenericCommand(flags StrSetType, key string, val *db.RedisObj, expire uint64, uint int) Reply {

	var (
		milliseconds int64
		ok           bool
	)

	if expire > 0 {
		ok, milliseconds = cmd.getExpireMillisecondsOrReply(expire, flags, uint)
		if !ok {
			return Reply{}
		}
	}

	_, exist := cmd.db.LookupKey(key)

	if (flags&ObjSetXX != 0 && !exist) || (flags&ObjSetNX != 0 && exist) {
		if !(flags&ObjSetGet != 0) {
			cmd.c.AddReply(SharedNullReply)
		}
		return
	}

}

func (cmd *StrCmd) getExpireMillisecondsOrReply(expire uint64, flags StrSetType, uint int) (bool, int64) {
}

func (cmd *StrCmd) getGenericCommand() {

}
