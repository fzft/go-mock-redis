package node

import (
	"github.com/fzft/go-mock-redis/db"
	"strconv"
)

/* ===================== Creation and parsing of objects ==================== */

func createObject(t db.ObjectType, ptr any) *db.RedisObj {
	return db.NewRedisObj(t, db.EncodingRaw, ptr, 0)
}

// stringObjectLen returns the length of the string object.
func stringObjectLen(o *db.RedisObj) int {
	return len([]rune(o.Value.(string)))
}

// createRawStringObject
// Create a string object with encoding OBJ_ENCODING_RAW
func createRawStringObject(ptr string) *db.RedisObj {
	return createObject(db.StringType, ptr)
}

// createEmbeddedStringObject
// Create a string object with encoding OBJ_ENCODING_EMBSTR
func createEmbeddedStringObject(ptr string) *db.RedisObj {
	return db.NewRedisObj(db.StringType, db.EncodingEmbStr, ptr, 0)
}

// getLongLongFromObject
func getLongLongFromObject(o *db.RedisObj) uint64 {
	if o == nil {
		return 0
	} else {
		if o.EncodingObject() {
			if v, ok := o.Value.(uint64); ok {
				return v
			} else {
				return 0
			}
		} else if o.Encoding == db.EncodingInt {
			return o.Value.(uint64)
		} else {
			panic("Unknown string encoding")
		}
	}
}

func ll2String(prefix byte, ll int64) []byte {
	// Convert int64 to string
	s := strconv.FormatInt(ll, 10)

	// Create a byte slice with size = 1 (for prefix) + len(s) + 2 (for '\r' and '\n')
	buf := make([]byte, 1+len(s)+2)

	// Set the prefix
	buf[0] = prefix

	// Copy the number string to buf
	copy(buf[1:], s)

	// Add '\r' and '\n' to the buffer
	buf[len(s)+1] = '\r'
	buf[len(s)+2] = '\n'

	return buf
}
