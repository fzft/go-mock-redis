package db

const (
	INITIAL_DB_SIZE = 16
)

type LookupType uint8

const (
	LookupNone      LookupType = 1 << iota
	LookupNoTouch              // Don't update LRU
	LookupNoNotify             // Don't trigger keyspace event on key misses
	LookupNoStats              // Don't update keyspace hits/misses counters.
	LookupWrite                // Delete expired keys even in replicas.
	LookupNoExpire             // Avoid deleting lazy expired keys.
	LookupNoEffects = LookupNoNotify | LookupNoStats | LookupNoTouch | LookupNoExpire
)

type SetKeyType uint8

const (
	SetKeyKeepTTL SetKeyType = 1 << iota
	SetKeyNoSignal
	SetKeyAlreadyExists
	SetKeyDoesNotExist
	SetKeyAddOrUpdated
)

// RedisDb represents a Redis database
type RedisDb struct {
	dict   *HashTable[string, *RedisObj] // the keyspace for this DB
	expire *HashTable[string, uint64]    // timeout of keys with a timeout set
	id     uint64                        // database ID
	avgTTL uint64                        // average TTL, just for stats

	//Metric
	StatKeySpaceHits   uint64
	StatKeySpaceMisses uint64
}

func New(id uint64) *RedisDb {
	return &RedisDb{
		id:     id,
		dict:   NewHashTable[string, *RedisObj](INITIAL_DB_SIZE),
		expire: NewHashTable[string, uint64](INITIAL_DB_SIZE),
	}
}

// SetKey sets the key to the value
// High level Set operation. this function can be used in order to set a key. whatever it was existing or not, to a new object
// 1. TODO
// 2.
//  3. The expire time of the key is reset (the key is made persistent)
//     unless 'SETKEY_KEEPTTL' is enabled in flags.
//  4. The key lookup can take place outside this interface outcome will be
//     *    delivered with 'SETKEY_ALREADY_EXIST' or 'SETKEY_DOESNT_EXIST'
//
// All the new keys in the database should be created via this interface.
func (db *RedisDb) SetKey(key string, val *RedisObj, flags SetKeyType) {
	var keyFound int8

	switch {
	case flags&SetKeyAlreadyExists != 0:
		keyFound = 1
	case flags&SetKeyAddOrUpdated != 0:
		keyFound = -1
	case flags&SetKeyDoesNotExist == 0:
		_, exist := db.LookupKeyWrite(key)
		if exist {
			keyFound = 1
		}
	}

	switch keyFound {
	case 0:
		db.add(key, val)
	case -1:
		db.addInternal(key, val, true)
	case 1:
		db.setValue(key, val, true, nil)
	}

	if flags&SetKeyKeepTTL == 0 {
		db.RmExpire(key)
	}
	// TODO: signal
}

// GetExpire returns the expire time of the key
func (db *RedisDb) GetExpire(key string) int64 {

	// if the key does not exist or has no associated expire, return -1
	if db.expire.Empty() {
		return -1
	}

	when, exist := db.expire.Get(key)
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

// GenericDelete deletes the key from the dict and the expire dict
func (db *RedisDb) GenericDelete(key string) bool {
	// delete the key from the dict
	exist := db.dict.Delete(key)
	if !exist {
		return false
	}

	// delete the key from the expire dict
	db.expire.Delete(key)

	return true
}

func (db *RedisDb) LookupKeyWrite(key string) (*RedisObj, bool) {
	return db.lookupKeyWriteWithFlags(key, LookupNone)
}

func (db *RedisDb) lookupKeyWriteWithFlags(key string, flags LookupType) (*RedisObj, bool) {
	return db.lookupKey(key, flags|LookupWrite)
}

/* Lookup a key for read or write operations, or return NULL if the key is not
 * found in the specified DB. This function implements the functionality of
 * lookupKeyRead(), lookupKeyWrite() and their ...WithFlags() variants.
 *
 * Side-effects of calling this function:
 *
 * 1. A key gets expired if it reached it's TTL.
 * 2. The key's last access time is updated.
 * 3. The global keys hits/misses stats are updated (reported in INFO).
 * 4. If keyspace notifications are enabled, a "keymiss" notification is fired.
 *
 * Flags change the behavior of this command:
 *
 *  LookupNone (or zero): No special flags are passed.
 *  LookupNoTouch: Don't alter the last access time of the key.
 *  LookupNoNotify: Don't trigger keyspace event on key miss.
 *  LookupNoStats: Don't increment key hits/misses counters.
 *  LookupWrite: Prepare the key for writing (delete expired keys even on
 *                replicas, use separate keyspace stats and events (TODO)).
 *  LookupNoExpire: Perform expiration check, but avoid deleting the key,
 *                   so that we don't have to propagate the deletion.
 */

func (db *RedisDb) lookupKey(key string, flags LookupType) (*RedisObj, bool) {
	val, exist := db.dict.Get(key)
	if exist {
		// update the access time for the aging algorithm
		if flags&LookupNoTouch == 0 {
			val.LRU = lruClock(100, 100)
		}

		if flags&(LookupNoStats|LookupWrite) == 0 {
			db.StatKeySpaceHits++
		}
	} else {
		if flags&(LookupNoStats|LookupWrite) == 0 {
			db.StatKeySpaceMisses++
		}
	}

	return val, exist
}

func (db *RedisDb) add(key string, val *RedisObj) {
	db.addInternal(key, val, false)
}

// addInternal adds the key to the dict
func (db *RedisDb) addInternal(key string, val *RedisObj, updateIfExist bool) {
	de, exist := db.dict.AddRaw(key)
	if exist && updateIfExist {
		db.setValue(key, val, true, de)
		return
	}
	initObjectLRUOrLFU(val)
	db.dict.SetVal(de, val)
	// TODO: notifyKeyspaceEvent
}

func (db *RedisDb) setValue(key string, val *RedisObj, overwrite bool, entry *Entry[string, *RedisObj]) {
	if entry == nil {
		entry, _ = db.dict.GetEntry(key)
	}
	old := db.dict.GetVal(entry)

	val.LRU = old.LRU

	if overwrite {
		// TODO:
	}

	db.dict.SetVal(entry, val)
}
