package db

import (
	"fmt"
	"hash/fnv"
	"math/rand"
	"reflect"
)

const (
	loadFactor = 0.7
)

type SliceFactory[K any] func(capacity int) []K

type Entry[K any, V any] struct {
	Key   K
	Value V
	Next  *Entry[K, V]
}

type HashTable[K any, V any] struct {
	Table []*Entry[K, V]
	Size  int
	Count int
}

func NewHashTable[K any, V any](initSize int) *HashTable[K, V] {
	return &HashTable[K, V]{
		Table: make([]*Entry[K, V], initSize),
		Size:  initSize,
	}
}

func (h *HashTable[K, V]) Hash(key K) int {
	keyString := fmt.Sprintf("%v", key)
	hasher := fnv.New32a()
	hasher.Write([]byte(keyString))
	return int(hasher.Sum32()) % h.Size
}

func (h *HashTable[K, V]) Set(key K, value V) {
	// Check if we need to resize the hash table
	if float64(h.Count)/float64(h.Size) > loadFactor {
		h.resize()
	}

	index := h.Hash(key)
	entry := &Entry[K, V]{Key: key, Value: value, Next: nil}
	if h.Table[index] == nil {
		h.Table[index] = entry
		h.Count++
	} else {
		curr := h.Table[index]
		for curr != nil {
			if reflect.DeepEqual(curr.Key, key) {
				curr.Value = value // Update the value
				return
			}
			if curr.Next == nil {
				curr.Next = entry
				h.Count++
				return
			}
			curr = curr.Next
		}
	}
}

func (h *HashTable[K, V]) resize() {
	newSize := h.Size * 2
	newTable := make([]*Entry[K, V], newSize)
	oldTable := h.Table
	h.Table = newTable
	h.Size = newSize
	h.Count = 0 // Reset count because we'll be re-adding the elements

	for _, entry := range oldTable {
		for entry != nil {
			h.Set(entry.Key, entry.Value)
			entry = entry.Next
		}
	}
}

func (h *HashTable[K, V]) Delete(key K) bool {
	index := h.Hash(key)
	if h.Table[index] == nil {
		// The key doesn't exist
		return false
	}

	// Special case: check if the key matches the first entry in the list
	if reflect.DeepEqual(h.Table[index].Key, key) {
		h.Table[index] = h.Table[index].Next
		return true
	}

	prev := h.Table[index]
	curr := prev.Next
	for curr != nil {
		if reflect.DeepEqual(curr.Key, key) {
			prev.Next = curr.Next // Bypass the entry to be deleted
			return true
		}
		prev = curr
		curr = curr.Next
	}

	// If we reach here, the key wasn't found in the list
	return false
}

func (h *HashTable[K, V]) Get(key K) (V, bool) {
	index := h.Hash(key)
	curr := h.Table[index]
	for curr != nil {
		if reflect.DeepEqual(curr.Key, key) {
			return curr.Value, true
		}
		curr = curr.Next
	}
	var zero V
	return zero, false
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
func (h *HashTable[K, V]) GetSomeKeys(count int, factory SliceFactory[K]) []K {
	if h.Empty() {
		return nil
	}

	if count > h.Len() {
		count = h.Len()
	}

	keys := factory(count)
	visited := 0

	// This loop ensures that we've sampled enough keys or visited all the buckets.
	for len(keys) < count && visited < h.Size {
		index := rand.Intn(h.Size)
		curr := h.Table[index]
		if curr != nil {
			for curr != nil && len(keys) < count {
				keys = append(keys, curr.Key)
				curr = curr.Next
			}
		}
		visited++
	}

	return keys
}
