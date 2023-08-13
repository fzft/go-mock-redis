package db

const (
	INITIAL_DB_SIZE = 16
)

// RedisDb represents a Redis database
type RedisDb struct {
	dict   *HashTable[string, *RedisObj] // the keyspace for this DB
	expire *HashTable[string, uint64]    // timeout of keys with a timeout set
	id     uint64                        // database ID
	avgTTL uint64                        // average TTL, just for stats
}

func New(id uint64) *RedisDb {
	return &RedisDb{
		id:     id,
		dict:   NewHashTable[string, *RedisObj](INITIAL_DB_SIZE),
		expire: NewHashTable[string, uint64](INITIAL_DB_SIZE),
	}
}

// GetExpire returns the expire time of the key
func (db *RedisDb) GetExpire(obj *RedisObj) int64 {

	// if the key does not exist or has no associated expire, return -1
	if db.expire.Empty() {
		return -1
	}

	when, exist := db.expire.Get(obj.Key)
	if !exist {
		return -1
	}

	return int64(when)
}

// SetExpire sets the expire time of the key
func (db *RedisDb) SetExpire(key string, expire uint64) {

	// if the key does not exist, return
	_, exist := db.dict.Get(key)
	if !exist {
		return
	}

	db.expire.Set(key, expire)
}

// RmExpire removes the expire time of the key
func (db *RedisDb) RmExpire(key string) {

	// if the key does not exist, return
	_, exist := db.dict.Get(key)
	if !exist {
		return
	}

	db.expire.Delete(key)
}
