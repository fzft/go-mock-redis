package db

// https://www.eecg.toronto.edu/~enright/teaching/ece243S/notes/l26-caches.html
// the Redis using an approximated LRU algorithm

const LRU_BITS = 24

// LRU clock resolution in ms

const LRU_CLOCK_RESOLUTION = 1000
const LRU_CLOCK_MAX = ((1 << LRU_BITS) - 1)

const EVPOOL_SIZE = 4
const MEMORY_SAMPLES = 5

/* ----------------------------------------------------------------------------
 * Implementation of eviction, aging and LRU
 * --------------------------------------------------------------------------*/

type LRU struct {
	ep [EVPOOL_SIZE]evictionPoolEntry
	eh EvictorHelper
	db *RedisDb
}

func NewLRU(db *RedisDb) *LRU {
	var ep [EVPOOL_SIZE]evictionPoolEntry
	for i := 0; i < EVPOOL_SIZE; i++ {
		ep[i] = evictionPoolEntry{}
	}
	return &LRU{
		ep: ep,
		eh: EvictorHelper{},
		db: db,
	}
}

// lruClock obtain the current LRU clock
// if the current resolution lower than the frequency we refresh the
// LRU clock we return the precomputed value,otherwise we need to resort to a system call
func lruClock(hz int, srvClock int64) int64 {
	var lruClock int64
	if 1000/hz <= LRU_CLOCK_RESOLUTION {
		lruClock = srvClock
	} else {
		lruClock = getLRUClock()
	}
	return lruClock
}

func getLRUClock() int64 {
	return (mstime() / LRU_CLOCK_RESOLUTION) & LRU_CLOCK_MAX
}

// estimateObjectIdleTime given an object returns the min number of milliseconds the object was never
// requested, using an approximated LRU algorithm.
func (lru *LRU) estimateObjectIdleTime(hz int, srvClock int64, o *RedisObj) int64 {
	lruClock := lruClock(hz, srvClock)
	if lruClock >= o.LRU {
		return (lruClock - o.LRU) * LRU_CLOCK_RESOLUTION
	} else {
		return (lruClock + (LRU_CLOCK_MAX - o.LRU)) * LRU_CLOCK_RESOLUTION
	}
}

/* LRU approximation algorithm
 *
 * Redis uses an approximation of the LRU algorithm that runs in constant
 * memory. Every time there is a key to expire, we sample N keys (with
 * N very small, usually in around 5) to populate a pool of best keys to
 * evict of M keys (the pool size is defined by EVPOOL_SIZE).
 *
 * The N keys sampled are added in the pool of good keys to expire (the one
 * with an old access time) if they are better than one of the current keys
 * in the pool.
 *
 * After the pool is populated, the best key we have in the pool is expired.
 * However note that we don't remove keys from the pool when they are deleted
 * so the pool may contain keys that no longer exist.
 *
 * When we try to evict a key, and all the entries in the pool don't exist
 * we populate it again. This time we'll be sure that the pool has at least
 * one key that can be evicted, if there is at least one key that can be
 * evicted in the whole database. */

// EvictionPoolPopulate  this is a helper function for the eviction
// We insert keys on place in ascending idle time on the left, and keys
// with the same idle time are put on the right
func (lru *LRU) EvictionPoolPopulate(sampleDict *HashTable[string, *RedisObj], keyDict *HashTable[string, *RedisObj]) {

	keys := sampleDict.GetSomeKeys(MEMORY_SAMPLES)

	for _, key := range keys {
		o, _ := keyDict.Get(key)
		idle := lru.estimateObjectIdleTime(100, 100, o)
		// Find the first empty slot or the first slot that has a lower idle time than the current key.
		k := 0
		for k < EVPOOL_SIZE && lru.ep[k].Key != "" && lru.ep[k].Idle < idle {
			k++
		}
		if k == 0 && lru.ep[EVPOOL_SIZE-1].Key != "" {
			// Can't insert if the element is < the worst element we have and there are no empty slots.
			continue
		} else if k < EVPOOL_SIZE && lru.ep[k].Key == "" {
			// Found an empty slot.
		} else {
			// Inserting in the middle.
			if lru.ep[EVPOOL_SIZE-1].Key == "" {
				// There is an empty slot at the end, shift all slots from k to the end to the right.
				copy(lru.ep[k+1:], lru.ep[k:])
			} else {
				/* No free space on right? Insert at k-1 */
				k--
				/* Shift all elements on the left of k (included) to the
				 * left, so we discard the element with smaller idle time. */

				copy(lru.ep[:k], lru.ep[1:k+1])

			}
		}
		lru.ep[k].Key = key
		lru.ep[k].Idle = idle
	}
}

// PerformEvictions ...
func (lru *LRU) PerformEvictions() (result EvictStatus) {
	var keysFreed int64

	memoryStatus, ok := lru.eh.getMaxmemoryState()
	if ok {
		result = EvictOk
		return
	}

	memFreed := int64(0)
	for memFreed < memoryStatus.ToFree {
		var (
			bestkey string
			exist   bool
		)
		for bestkey == "" {
			keys := lru.db.dict.Len()
			if keys == 0 {
				break // no more keys to evict
			}

			/* Go backward from best to worst element to evict. */
			for k := EVPOOL_SIZE - 1; k >= 0; k-- {
				if lru.ep[k].Key == "" {
					continue
				}

				_, exist = lru.db.expire.Get(lru.ep[k].Key)
				// If the key exists in the expire set, we can try to evict it.
				if exist {
					bestkey = lru.ep[k].Key
					break
				} else {
					//  Ghost... Iterate again
				}
			}
		}
		if bestkey != "" {
			/* We compute the amount of memory freed by db*Delete() alone.
			 * It is possible that actually the memory needed to propagate
			 * the DEL in AOF and replication link is greater than the one
			 * we are freeing removing the key, but we can't account for
			 * that otherwise we would never exit the loop.
			 *
			 * Same for CSC invalidation messages generated by signalModifiedKey.
			 *
			 * AOF and Output buffer memory will be freed eventually so
			 * we only care about memory used by the key space. */
			delta := getUsedMemory()
			lru.db.GenericDelete(bestkey)
			delta -= getUsedMemory()
			memFreed += delta
			keysFreed += 1
		}
	}

	return EvictOk
}
