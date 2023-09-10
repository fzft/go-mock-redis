package node

import (
	"github.com/fzft/go-mock-redis/db"
	"strings"
	"unsafe"
)

type UserFlag uint8

const (
	UserFlagEnabled UserFlag = 1 << iota
	UserFlagDisabled
	UserFlagNoPass
	UserFlagSkipSanitizePayload
	UserFlagSanitizePayload
)

type User struct {
	name      string
	flags     UserFlag
	passwords *db.List[string] // List of password hashes
	selectors *db.List[string] /* A list of selectors this user validates commands
	   against. This list will always contain at least
	   one selector for backwards compatibility. */
	aclString *db.RedisObj // cached acl string
}

// ACLCheckAllUserCommandPerm low level api that checks if a specified user is able to execute a command.
func (u *User) ACLCheckAllUserCommandPerm(cmd RedisCommand, argv []*db.RedisObj) {
	iter := u.selectors.NewListIterator(db.DIRECTION_HEAD)

	for node := iter.NextNode(); node != nil; node = iter.NextNode() {
	}
}

type AclCheckAllPerm uint8

const (
	ACLOK AclCheckAllPerm = iota
	ACLDeniedCmd
	ACLDeniedKey
	ACLDeniedAuth    // Only used for ACL LOG entries.
	ACLDeniedChannel // Only used for pub/sub commands.
)

type SelectorFlag uint8

const (
	SelectorFlagAllKeys SelectorFlag = 1 << iota
	SelectorFlagAllCommands
	SelectorFlagAllChannels
)

type ACLSelectorFlags struct {
	name string
	flag SelectorFlag
}

var AclSelectorFlags = []ACLSelectorFlags{
	{"allkeys", SelectorFlagAllKeys},
	{"allcommands", SelectorFlagAllCommands},
	{"allchannels", SelectorFlagAllChannels},
	{name: "", flag: 0}, // Terminator
}

// aclSelector are private and not exposed outside
type aclSelector struct {
	flags SelectorFlag
	/* allowedCommands The bit in allowedCommands is set if this user has the right to
	 * execute this command.
	 *
	 * If the bit for a given command is NOT set and the command has
	 * allowed first-args, Redis will also check allowed_firstargs in order to
	 * understand if the command can be executed. */
	allowedCommands [(UserCommandBitsCount + 63) / 64]uint64
	/* allowedFirstArgs is used by ACL rules to block access to a command unless a
	 * specific argv[1] is given.
	 *
	 * For each command ID (corresponding to the command bit set in allowed_commands),
	 * This array points to an array of SDS strings, terminated by a NULL pointer,
	 * with all the first-args that are allowed for this command. When no first-arg
	 * matching is used, the field is just set to NULL to avoid allocating
	 * USER_COMMAND_BITS_COUNT pointers. */
	allowedFirstArgs [][]string

	patterns *db.List[string] // List of patterns this user validates commands against.

	channels *db.List[string] // List of channels this user can access in pub/sub.
	/* A string representation of the ordered categories and commands, this
	 * is used to regenerate the original ACL string for display. */
	cmdRules string
}

func (s *aclSelector) checkCmd(cmd RedisCommand, argv []*db.RedisObj, argc int) AclCheckAllPerm {
	if s.flags&SelectorFlagAllCommands == 0 && cmd.Flags()&CmdNoAuth == 0 {
		// if the bit is not set we have to check further, in case the command is allowed just with specific first args.

		id := cmd.Id()

		if !s.cmdBit(id) {
			// check if the first argument is allowed.
			if argc < 2 || s.allowedFirstArgs == nil || s.allowedFirstArgs[id] == nil {
				return ACLDeniedCmd
			}

			var subid uint64
			for {
				if s.allowedFirstArgs[id][subid] == "" {
					return ACLDeniedCmd
				}

				var idx int
				if cmd.Parent() != nil {
					idx = 2
				} else {
					idx = 1
				}

				if !strings.EqualFold(argv[idx].Value.(string), s.allowedFirstArgs[id][subid]) {
					break
				}
				subid++
			}

		}
	}

	/* Check if the user can execute commands explicitly touching the keys
	 * mentioned in the command arguments. */
	//if s.flags&SelectorFlagAllKeys == 0 {
	//
	//}
	return ACLOK
}

// cmdBit check if the specified command bit is set for the specified user.
func (s *aclSelector) cmdBit(id uint64) bool {
	if word, bit, ok := ACLGetCommandBitCoordinates(id); ok {
		return s.allowedCommands[word]&bit != 0
	}
	return false
}

type ACLCategoryItem struct {
	name string
	flag uint64
}

const (
	ACLCategoryKeyspace = 1 << iota
	ACLCategoryRead
	ACLCategoryWrite
	ACLCategorySet
	ACLCategorySortedSet
	ACLCategoryList
	ACLCategoryHash
	ACLCategoryString
	ACLCategoryBitmap
	ACLCategoryHyperLogLog
	ACLCategoryGeo
	ACLCategoryStream
	ACLCategoryPubSub
	ACLCategoryAdmin
	ACLCategoryFast
	ACLCategorySlow
	ACLCategoryBlocking
	ACLCategoryDangerous
	ACLCategoryConnection
	ACLCategoryTransaction
	ACLCategoryScripting
)

var ACLCommandCategories = []ACLCategoryItem{
	{"keyspace", ACLCategoryKeyspace},
	{"read", ACLCategoryRead},
	{"write", ACLCategoryWrite},
	{"set", ACLCategorySet},
	{"sortedset", ACLCategorySortedSet},
	{"list", ACLCategoryList},
	{"hash", ACLCategoryHash},
	{"string", ACLCategoryString},
	{"bitmap", ACLCategoryBitmap},
	{"hyperloglog", ACLCategoryHyperLogLog},
	{"geo", ACLCategoryGeo},
	{"stream", ACLCategoryStream},
	{"pubsub", ACLCategoryPubSub},
	{"admin", ACLCategoryAdmin},
	{"fast", ACLCategoryFast},
	{"slow", ACLCategorySlow},
	{"blocking", ACLCategoryBlocking},
	{"dangerous", ACLCategoryDangerous},
	{"connection", ACLCategoryConnection},
	{"transaction", ACLCategoryTransaction},
	{"scripting", ACLCategoryScripting},
	{name: "", flag: 0}, // Terminator
}

type ACLUserFlag struct {
	name string
	flag UserFlag
}

var ACLUserFlags = []ACLUserFlag{
	{"on", UserFlagEnabled},
	{"off", UserFlagDisabled},
	{"nopass", UserFlagNoPass},
	{"skip-sanitize-payload", UserFlagSkipSanitizePayload},
	{"sanitize-payload", UserFlagSanitizePayload},
	{name: "", flag: 0}, // Terminator
}

type Acl struct {
	CommandId db.RaxTree[int]
}

func (c *Client) ACLCheckAllPerm() {

}

func ACLGetCommandBitCoordinates(id uint64) (word uint64, bit uint64, ok bool) {
	if id >= UserCommandBitsCount {
		return 0, 0, false
	}

	word = id / (uint64(unsafe.Sizeof(uint64(0))) * 8)
	bit = 1 << (id % (uint64(unsafe.Sizeof(uint64(0))) * 8))

	return word, bit, true
}
