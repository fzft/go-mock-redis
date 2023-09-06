package node

import (
	"github.com/fzft/go-mock-redis/db"
	"strings"
)

const (
	UintSeconds = iota
	UintMilliseconds
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

type CommandType uint8

const (
	CommandSet CommandType = iota
	CommandGet
)

// StrCmd handles string commands.
type StrCmd struct {
	c  *Client
	db *db.RedisDb
}

// NewStrCmd returns a new StrCmd.
func NewStrCmd(c *Client, db *db.RedisDb) *StrCmd {
	return &StrCmd{c: c, db: db}
}

// Set implements the SET command.
func (cmd *StrCmd) Set() {
	flags := ObjNoFlags
	retFlags, expire, uint, ok := cmd.parseExtendedStringArgumentsOrReply(flags, CommandSet)
	if !ok {
		return
	}
	cmd.setGenericCommand(retFlags, cmd.c.argv[1].Value.(string), cmd.c.argv[2], expire, uint)
}

// SetNx implements the SETNX command.
func (cmd *StrCmd) SetNx() {
	cmd.setGenericCommand(ObjSetNX, cmd.c.argv[1].Value.(string), cmd.c.argv[2], nil, 0)
}

// SetEx implements the SETEX command.
func (cmd *StrCmd) SetEx() {
	cmd.setGenericCommand(ObjSetEX, cmd.c.argv[1].Value.(string), cmd.c.argv[3], cmd.c.argv[2], UintSeconds)
}

// PSetEx implements the PSETEX command.
func (cmd *StrCmd) PSetEx() {
	cmd.setGenericCommand(ObjSetPX, cmd.c.argv[1].Value.(string), cmd.c.argv[3], cmd.c.argv[2], UintMilliseconds)
}

// Get implements the GET command.
func (cmd *StrCmd) Get() {
	cmd.getGenericCommand()
}

func (cmd *StrCmd) getGenericCommand() bool {
	o, exist := cmd.db.LookupKeyRead(cmd.c.argv[1].Value.(string))
	if !exist {
		cmd.c.AddReply(SharedNull3)
		return false
	}
	cmd.c.AddReplyBulk(o)
	return true
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
func (cmd *StrCmd) setGenericCommand(flags StrSetType, key string, val *db.RedisObj, expire *db.RedisObj, uint int) {

	var (
		milliseconds uint64
		ok           bool
		setkeyFlags  db.SetKeyType
	)

	if expire != nil {
		milliseconds, ok = cmd.getExpireMillisecondsOrReply(expire, flags, uint)
		if !ok {
			return
		}
	}

	_, exist := cmd.db.LookupKeyWrite(key)

	if (flags&ObjSetXX != 0 && !exist) || (flags&ObjSetNX != 0 && exist) {
		if !(flags&ObjSetGet != 0) {
			cmd.c.AddReply(SharedNull3)
		}
		return
	}

	/* When expire is not NULL, we avoid deleting the TTL so it can be updated later instead of being deleted and then created again. */
	if (flags&ObjSetKeepTTL) != 0 || expire == nil {
		setkeyFlags |= db.SetKeyKeepTTL
	} // We don't set setkeyFlags to 0 in an else, because it might have other bits set previously.

	if exist {
		setkeyFlags |= db.SetKeyAlreadyExists
	} else {
		setkeyFlags |= db.SetKeyDoesNotExist
	}

	cmd.db.SetKey(key, val, setkeyFlags)
	// TODO: notifyKeyspaceEvent(NOTIFY_STRING, "set", key, cmd.db.GetID())

	if expire != nil {
		cmd.db.SetExpire(key, milliseconds)
		// TODO: notifyKeyspaceEvent(NOTIFY_GENERIC, "expire", key, cmd.db.GetID())
	}

	if flags&ObjSetGet == 0 {
		cmd.c.AddReply(SharedOk)
	}
}

/*
 * The parseExtendedStringArgumentsOrReply() function performs the common validation for extended
 * string arguments used in SET and GET command.
 *
 * Get specific commands - PERSIST/DEL
 * Set specific commands - XX/NX/GET
 * Common commands - EX/EXAT/PX/PXAT/KEEPTTL
 *
 * Function takes pointers to client, flags, unit, pointer to pointer of expire obj if needed
 * to be determined and command_type which can be COMMAND_GET or COMMAND_SET.
 *
 * If there are any syntax violations C_ERR is returned else C_OK is returned.
 *
 * Input flags are updated upon parsing the arguments. Unit and expire are updated if there are any
 * EX/EXAT/PX/PXAT arguments. Unit is updated to millisecond if PX/PXAT is set.
 */
func (cmd *StrCmd) parseExtendedStringArgumentsOrReply(flags StrSetType, commandType CommandType) (retFlags StrSetType, expire *db.RedisObj, uint int, ok bool) {
	var j int
	if commandType == CommandGet {
		j = 2
	} else {
		j = 3
	}

	defer func() {
		retFlags = flags
	}()

	for ; j < cmd.c.argc; j++ {
		var (
			next *db.RedisObj
			opt  string
		)
		opt, ok = cmd.c.argv[j].Value.(string)
		if !ok {
			cmd.c.AddReply(SharedSyntaxErr)
			return
		}
		if j == cmd.c.argc-1 {
			next = nil
		} else {
			next = cmd.c.argv[j+1]
		}

		fmtOpt := strings.ToUpper(opt)
		if fmtOpt == "NX" && flags&ObjSetXX == 0 && commandType == CommandSet {
			flags |= ObjSetNX
		} else if fmtOpt == "XX" && flags&ObjSetNX == 0 && commandType == CommandSet {
			flags |= ObjSetXX
		} else if fmtOpt == "GET" && commandType == CommandSet {
			flags |= ObjSetGet
		} else if strings.EqualFold(opt, "KEEPTTL") &&
			flags&ObjPERSIST == 0 &&
			flags&ObjSetEX == 0 &&
			flags&ObjEXAT == 0 &&
			flags&ObjSetPX == 0 &&
			flags&ObjPXAT == 0 &&
			commandType == CommandSet {
			flags |= ObjSetKeepTTL
		} else if strings.EqualFold(opt, "PERSIST") &&
			flags&ObjSetEX == 0 &&
			flags&ObjEXAT == 0 &&
			flags&ObjSetPX == 0 &&
			flags&ObjPXAT == 0 &&
			flags&ObjSetKeepTTL == 0 &&
			commandType == CommandGet {
			flags |= ObjPERSIST
		} else if fmtOpt == "EX" &&
			flags&ObjSetKeepTTL == 0 &&
			flags&ObjPERSIST == 0 &&
			flags&ObjEXAT == 0 &&
			flags&ObjPXAT == 0 &&
			flags&ObjSetPX == 0 &&
			next != nil {
			flags |= ObjSetEX
			expire = next
			j++
		} else if fmtOpt == "PX" &&
			flags&ObjSetKeepTTL == 0 &&
			flags&ObjPERSIST == 0 &&
			flags&ObjSetEX == 0 &&
			flags&ObjEXAT == 0 &&
			flags&ObjPXAT == 0 &&
			next != nil {
			flags |= ObjSetPX
			uint = UintMilliseconds
			expire = next
			j++
		} else if fmtOpt == "EXAT" &&
			flags&ObjSetKeepTTL == 0 &&
			flags&ObjPERSIST == 0 &&
			flags&ObjSetEX == 0 &&
			flags&ObjPXAT == 0 &&
			flags&ObjSetPX == 0 &&
			next != nil {
			flags |= ObjEXAT
			expire = next
			j++
		} else if fmtOpt == "PXAT" &&
			flags&ObjSetKeepTTL == 0 &&
			flags&ObjPERSIST == 0 &&
			flags&ObjSetEX == 0 &&
			flags&ObjEXAT == 0 &&
			flags&ObjSetPX == 0 &&
			next != nil {
			flags |= ObjPXAT
			uint = UintMilliseconds
			expire = next
			j++
		} else {
			cmd.c.AddReply(SharedSyntaxErr)
			return
		}
	}
	ok = true
	return
}

// getExpireMillisecondsOrReply extracts the expire time in milliseconds from the expire obj.
func (cmd *StrCmd) getExpireMillisecondsOrReply(expire *db.RedisObj, flags StrSetType, uint int) (uint64, bool) {
	ret := getLongLongFromObject(expire)
	if ret == 0 {
		cmd.c.AddReplyError("invalid expire time in SETEX")
		return 0, false
	}
	if (uint == UintSeconds && ret > 9223372036) || (uint == UintMilliseconds && ret > 9223372036854775) {
		cmd.c.AddReplyErrorExpireTime()
		return 0, false
	}

	if uint == UintSeconds {
		ret *= 1000
	}

	return ret, true
}
