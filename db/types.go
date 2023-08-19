package db

type ObjectType uint8

const (
	StringType ObjectType = iota
	ListType
	SetType
	ZSetType
	HashType
	StreamType
)

type EncodingType int

const (
	EncodingRaw        EncodingType = iota // Raw encoding
	EncodingInt                            // Encoded as integer
	EncodingHT                             // Encoded as hash table
	EncodingZipMap                         // Encoded as zipmap
	EncodingLinkedList                     // Encoded as regular linked list
	EncodingZipList                        // Encoded as ziplist
	EncodingIntSet                         // Encoded as intset
	EncodingSkipList
	EncodingEmbStr
	EncodingQuickList
	EncodingStream
	EncodingListPack
)

// RedisObj is the basic object type in Redis
type RedisObj struct {
	Type     ObjectType
	Encoding EncodingType
	LRU      int64
	Value    any
}

func (ro *RedisObj) GetObjType() ObjectType {
	return ro.Type
}
