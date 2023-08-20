package db

import (
	"fmt"
	"hash/fnv"
	"math/rand"
	"reflect"
)

// hashtable provides a simple implementation of a hashtable with support for
// incremental rehashing.
//
// The HashTable data structure supports basic operations such as Get, Set, Delete, etc.
// To handle the case when the hash table becomes too full, it uses incremental rehashing
// to migrate entries to a new, larger table without causing a significant performance drop
// during the migration process.
//
// Important Notes:
// - The rehashStep method is accessed within both the FindPositionForInsert and Set methods.
//   This allows the HashTable to gradually migrate data to a larger table while still being
//   able to serve other requests.
//

const (
	loadFactor       = 0.7
	rehashingBuckets = 10 // Number of buckets to move during one rehashing step
)

type Entry[K any, V any] struct {
	Key   K
	Value V
	Next  *Entry[K, V]
}

type HashTable[K any, V any] struct {
	Table         []*Entry[K, V]
	Size          int
	RehashingIdx  int // -1 if not rehashing
	RehashingSize int // The size of the new table when rehashing
	RehashingTbl  []*Entry[K, V]
	Count         int
}

func NewHashTable[K any, V any](initSize int) *HashTable[K, V] {
	return &HashTable[K, V]{
		Table:        make([]*Entry[K, V], initSize),
		Size:         initSize,
		RehashingIdx: -1,
	}
}

func (h *HashTable[K, V]) Hash(key K, size int) int {
	keyString := fmt.Sprintf("%v", key)
	hasher := fnv.New32a()
	hasher.Write([]byte(keyString))
	return int(hasher.Sum32()) % size
}

func (h *HashTable[K, V]) Set(key K, value V) {

	defer func() {
		// Start rehashing if load factor exceeded
		if float64(h.Count)/float64(h.Size) > loadFactor {
			h.startRehashing()
		}
	}()

	// If we're rehashing, do one rehashing step
	if h.RehashingIdx >= 0 {

		// NOTE: rehashStep is accessed both in FindPositionForInsert and Set methods.
		h.rehashStep()

		// if we're still rehashing, insert into the new table
		if h.RehashingIdx >= 0 {
			// Check if the key exists in the new table
			newIndex := h.Hash(key, h.RehashingSize)
			if h.RehashingTbl[newIndex] == nil {
				entry := &Entry[K, V]{Key: key, Value: value, Next: nil}
				h.RehashingTbl[newIndex] = entry
				h.Count++
				h.IncreaseUsedMemory(key, value)
			} else {
				curr := h.RehashingTbl[newIndex]
				for curr != nil {
					if reflect.DeepEqual(curr.Key, key) {
						curr.Value = value
						return
					}
					if curr.Next == nil {
						entry := &Entry[K, V]{Key: key, Value: value, Next: nil}
						curr.Next = entry
						h.Count++
						h.IncreaseUsedMemory(key, value)
						return
					}
					curr = curr.Next
				}
			}
			return // We should return after handling the RehashingTbl to avoid double insertion
		}
	}

	index := h.Hash(key, h.Size)
	entry := &Entry[K, V]{Key: key, Value: value, Next: nil}
	if h.Table[index] == nil {
		h.Table[index] = entry
		h.Count++
		h.IncreaseUsedMemory(key, value)
	} else {
		curr := h.Table[index]
		for curr != nil {
			if reflect.DeepEqual(curr.Key, key) {
				h.IncreaseUsedMemory(key, value)
				curr.Value = value
				return
			}
			if curr.Next == nil {
				curr.Next = entry
				h.Count++
				h.IncreaseUsedMemory(key, value)
				return
			}
			curr = curr.Next
		}
	}

}

func (h *HashTable[K, V]) startRehashing() {
	if h.RehashingIdx < 0 { // Not already rehashing
		h.RehashingSize = h.Size * 2
		h.RehashingTbl = make([]*Entry[K, V], h.RehashingSize)
		h.RehashingIdx = 0
	}
}

// rehashStep moves rehashingBuckets buckets from the old table to the new table.
// in this step, the old table used memory will be decreased and the new table used memory will be increased.
func (h *HashTable[K, V]) rehashStep() {
	for i := 0; i < rehashingBuckets && h.RehashingIdx < h.Size; i++ {
		entries := h.Table[h.RehashingIdx]
		h.Table[h.RehashingIdx] = nil
		for entries != nil {
			next := entries.Next

			index := h.Hash(entries.Key, h.RehashingSize)
			entries.Next = h.RehashingTbl[index]
			h.RehashingTbl[index] = entries

			entries = next
		}

		h.RehashingIdx++
	}

	if h.RehashingIdx == h.Size {
		h.Table = h.RehashingTbl
		h.Size = h.RehashingSize
		h.RehashingTbl = nil
		h.RehashingIdx = -1
		h.RehashingSize = 0
	}
}

func (h *HashTable[K, V]) Delete(key K) bool {
	if h.RehashingIdx >= 0 {
		// If rehashing, try to delete from both tables.
		deletedFromOld := h.deleteFromTable(key, h.Table)
		deletedFromNew := h.deleteFromTable(key, h.RehashingTbl)
		return deletedFromOld || deletedFromNew
	} else {
		// If not rehashing, just delete from the main table.
		return h.deleteFromTable(key, h.Table)
	}
}

// Helper function to delete an entry from a specific table.
func (h *HashTable[K, V]) deleteFromTable(key K, table []*Entry[K, V]) bool {
	size := len(table)
	index := h.Hash(key, size)

	if table[index] == nil {
		// The key doesn't exist in this table.
		return false
	}

	// Special case: check if the key matches the first entry in the list.
	if reflect.DeepEqual(table[index].Key, key) {
		h.DecreaseUsedMemory(key, table[index].Value)
		table[index] = table[index].Next
		return true
	}

	prev := table[index]
	curr := prev.Next
	for curr != nil {
		if reflect.DeepEqual(curr.Key, key) {
			h.DecreaseUsedMemory(key, curr.Value)
			prev.Next = curr.Next // Bypass the entry to be deleted.
			return true
		}
		prev = curr
		curr = curr.Next
	}

	// If we reach here, the key wasn't found in this table's list.
	return false
}

func (h *HashTable[K, V]) Get(key K) (V, bool) {
	entry, exist := h.GetEntry(key)
	if exist {
		return entry.Value, true
	} else {
		var zero V
		return zero, false
	}
}

func (h *HashTable[K, V]) GetEntry(key K) (*Entry[K, V], bool) {
	// If rehashing is ongoing, check the new table first
	if h.RehashingIdx >= 0 {
		newIndex := h.Hash(key, h.RehashingSize)
		for curr := h.RehashingTbl[newIndex]; curr != nil; curr = curr.Next {
			if reflect.DeepEqual(curr.Key, key) {
				return curr, true
			}
		}
	}

	// Check the old table
	index := h.Hash(key, h.Size)
	for curr := h.Table[index]; curr != nil; curr = curr.Next {
		if reflect.DeepEqual(curr.Key, key) {
			return curr, true
		}
	}

	return nil, false
}

// AddRaw adds a raw entry to the hash table, without checking for duplicates or resizing.
// This is used for rehashing.
func (h *HashTable[K, V]) AddRaw(key K) (*Entry[K, V], bool) {
	return h.findPositionForInsert(key)
}

// findPositionForInsert finds the position within the hash table where the provided key should
// be inserted. If the key exists, it returns nil and sets the 'existing' entry pointer, if provided.
func (h *HashTable[K, V]) findPositionForInsert(key K) (*Entry[K, V], bool) {

	defer func() {
		if float64(h.Count)/float64(h.Size) > loadFactor {
			h.startRehashing()
		}
	}()

	entry := &Entry[K, V]{Key: key, Next: nil}

	if h.RehashingIdx >= 0 {
		// If rehashing is ongoing, perform a step before checking
		h.rehashStep()

		if h.RehashingIdx >= 0 {
			// Check the rehashing table
			newIndex := h.Hash(key, h.RehashingSize)
			if h.RehashingTbl[newIndex] == nil {
				h.RehashingTbl[newIndex] = entry
				h.Count++
				IncreaseUsedMemory(key)
			} else {
				// Check the chain for this hash slot
				prev := h.RehashingTbl[newIndex]
				for curr := prev; curr != nil; curr = curr.Next {
					if reflect.DeepEqual(curr.Key, key) {
						return curr, true // Key already exists in the rehashing table
					}
					if curr.Next == nil {
						curr.Next = entry
						h.Count++
						IncreaseUsedMemory(key)
					}
					prev = curr
				}
			}
		}
	}

	index := h.Hash(key, h.Size)
	if h.Table[index] == nil {
		h.Table[index] = entry
		h.Count++
		IncreaseUsedMemory(key)
	} else {
		curr := h.Table[index]
		for curr != nil {
			if reflect.DeepEqual(curr.Key, key) {
				return curr, true // Key already exists in the main table
			}
			if curr.Next == nil {
				curr.Next = entry
				h.Count++
				IncreaseUsedMemory(key)
			}
			curr = curr.Next
		}
	}
	return entry, false
}

// Helper function to retrieve an entry from a specific table.
func (h *HashTable[K, V]) getFromTable(key K, table []*Entry[K, V]) (V, bool) {
	size := len(table)
	index := h.Hash(key, size)

	curr := table[index]
	for curr != nil {
		if reflect.DeepEqual(curr.Key, key) {
			return curr.Value, true
		}
		curr = curr.Next
	}
	var zero V
	return zero, false
}

// SetVal ...
func (h *HashTable[K, V]) SetVal(entry *Entry[K, V], val V) {
	if entry == nil {
		return
	}
	entry.Value = val
	IncreaseUsedMemory(val)
}

// GetVal ...
func (h *HashTable[K, V]) GetVal(entry *Entry[K, V]) V {
	if entry == nil {
		var zero V
		return zero
	}
	return entry.Value
}

// Len returns the number of elements in the hash table
func (h *HashTable[K, V]) Len() int {
	return h.Count
}

// Empty returns true if the hash table is empty
func (h *HashTable[K, V]) Empty() bool {
	return h.Count == 0
}

// GetSomeKeys returns a slice of up to `count` keys sampled from the hash table.
// If the hash table has fewer than `count` keys, it returns all of them.
func (h *HashTable[K, V]) GetSomeKeys(count int) []K {
	if h.Empty() {
		return nil
	}

	if count > h.Len() {
		count = h.Len()
	}

	keys := make([]K, 0, count)
	visitedBuckets := make(map[int]struct{})

	// As long as we haven't got the required number of keys and haven't visited all buckets
	for len(keys) < count && len(visitedBuckets) < h.Size {
		index := rand.Intn(h.Size)

		// If this bucket is unvisited
		if _, alreadyVisited := visitedBuckets[index]; !alreadyVisited {
			visitedBuckets[index] = struct{}{} // Mark bucket as visited

			curr := h.Table[index]
			for curr != nil && len(keys) < count {
				keys = append(keys, curr.Key)
				curr = curr.Next
			}
		}
	}

	return keys
}

// IncreaseUsedMemory increases the used memory by the given amount
func (h *HashTable[K, V]) IncreaseUsedMemory(key K, val V) {
	IncreaseUsedMemory(key)
	IncreaseUsedMemory(val)
}

// DecreaseUsedMemory decreases the used memory by the given amount
func (h *HashTable[K, V]) DecreaseUsedMemory(key K, val V) {
	DecreaseUsedMemory(key)
	DecreaseUsedMemory(val)
}
