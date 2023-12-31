package db

// Redis maxmemory strategies
const (
	MAXMEMORY_FLAG_LRU = 1 << iota
	MAXMEMORY_FLAG_LFU
)

const (
	MAXMEMORY_VOLATILE_LRU = (0 << 8) | MAXMEMORY_FLAG_LRU
	MAXMEMORY_VOLATILE_LFU = (1 << 8) | MAXMEMORY_FLAG_LFU
	MAXMEMORY_VOLATILE_TTL = (2 << 8)
)

type EvictStatus int

const (
	EvictFail EvictStatus = iota
	EvictRunning
	EvictOk
)

// MemoryStatus is used to return information about the memory usage of the
type MemoryStatus struct {
	Total   int64   // total amount of bytes used.
	Logical int64   // the amount of memory used minus the slaves/AOF buffers.
	ToFree  int64   // the amount of memory that should be released
	Level   float64 // this usually ranges from 0 to 1, and reports the amount of
}

// evictionPoolEntry is used to store the key and idle time of a database key
type evictionPoolEntry struct {
	Key  string
	Idle int64
}

type Evictor interface {
	EvictionPoolPopulate(sampleDict *HashTable[string, *RedisObj], keyDict *HashTable[string, *RedisObj])
	PerformEvictions() EvictStatus
}

type EvictorHelper struct {
}

// getMaxmemoryState
// Returns the memory status of the system, including the total amount of
// memory used, the amount of memory that should be freed in order to return
func (eh EvictorHelper) getMaxmemoryState() (status MemoryStatus, ok bool) {

	memUsed := getUsedMemory()
	status.Total = memUsed

	status.Level = float64(memUsed) / float64(MAX_MEMORY)
	if memUsed <= MAX_MEMORY {
		return status, true
	}

	// Compute how much memory we need to free.
	memToFree := memUsed - MAX_MEMORY
	status.ToFree = memToFree

	return status, false
}

// initObjectLRUOrLFU
// TODO: LFU and SRV maxmemory strategies
func initObjectLRUOrLFU(o *RedisObj) {
	o.LRU = lruClock(100, 100)
}
