package commands

import "github.com/fzft/go-mock-redis/db"

type Reply interface {
	Content() any
	Type() db.ObjectType
	Encoding() []byte
}

type BaseReply struct {
	content string
}

func (r BaseReply) Content() any {
	return r.content
}

func (r BaseReply) Encoding() []byte {
	switch r.Type() {
	case db.StringType:
		return []byte(r.Content().(string))
	default:
		panic("not implemented")
	}
}

func (r BaseReply) Type() db.ObjectType {
	panic("not implemented")
}

type OkReply struct {
	BaseReply
}

func (r OkReply) Type() db.ObjectType {
	return db.StringType
}

type NullReply struct {
	BaseReply
}

func (r NullReply) Type() db.ObjectType {
	return db.StringType
}

type AbortReply struct {
	BaseReply
}

func (r AbortReply) Type() db.ObjectType {
	return db.StringType
}

var (
	SharedOkReply    = OkReply{BaseReply{content: "OK"}}
	SharedAbortReply = AbortReply{BaseReply{content: "ABORT"}}
	SharedNullReply  = NullReply{BaseReply{content: "NULL"}}
)

type Command interface {
	Set(key string, val *db.RedisObj, expire int64, uint int) Reply
	GetExpireMillisecondsOrReply(key string) (int64, bool)
}
