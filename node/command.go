package node

import "github.com/fzft/go-mock-redis/db"

type Reply interface {
	Content() any
	Encoding() db.EncodingType
	EncodingObject() bool
	Marshal() []byte
}

type BaseReply struct {
	content  string
	encoding db.EncodingType
}

func (r BaseReply) Content() any {
	return r.content
}

func (r BaseReply) Encoding() db.EncodingType {
	return r.encoding
}

func (r BaseReply) Marshal() []byte {
	switch r.encoding {
	case db.EncodingRaw, db.EncodingEmbStr:
		return []byte(r.Content().(string))
	default:
		panic("not implemented")
	}
}

func (r BaseReply) EncodingObject() bool {
	return r.encoding == db.EncodingRaw || r.encoding == db.EncodingEmbStr
}

type OkReply struct {
	BaseReply
}

type NullReply struct {
	BaseReply
}

type AbortReply struct {
	BaseReply
}

var (
	SharedOkReply    = OkReply{BaseReply{content: "OK", encoding: db.EncodingRaw}}
	SharedAbortReply = AbortReply{BaseReply{content: "ABORT", encoding: db.EncodingRaw}}
	SharedNullReply  = NullReply{BaseReply{content: "NULL", encoding: db.EncodingRaw}}
)

type Command interface {
	Set(key string, val *db.RedisObj, expire int64, uint int) Reply
	GetExpireMillisecondsOrReply(key string) (int64, bool)
}
