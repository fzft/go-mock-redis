package node

import (
	"fmt"
	"github.com/fzft/go-mock-redis/db"
)

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
	flag uint64
}

const (
	UserFlagEnabled = 1 << iota
	UserFlagDisabled
	UserFlagNoPass
	UserFlagSkipSanitizePayload
	UserFlagSanitizePayload
)

var ACLUserFlags = []ACLUserFlag{
	{"on", UserFlagEnabled},
	{"off", UserFlagDisabled},
	{"nopass", UserFlagNoPass},
	{"skip-sanitize-payload", UserFlagSkipSanitizePayload},
	{"sanitize-payload", UserFlagSanitizePayload},
	{name: "", flag: 0}, // Terminator
}

func main() {
	for _, category := range ACLCommandCategories {
		if category.name == "" {
			break // Terminator
		}
		fmt.Printf("Name: %s, Flag: %d\n", category.name, category.flag)
	}
}

type Acl struct {
	CommandId db.RaxTree[int]
}
