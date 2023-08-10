package db

import (
	"hash/fnv"
)

const (
	loadFactor = 0.7
)

type Entry[T any] struct {
	Key   string
	Value T
	Next  *Entry[T]
}

type HashTable[T any] struct {
	Table []*Entry[T]
	Size  int
	Count int
}

func NewHashTable[T any](initSize int) *HashTable[T] {
	return &HashTable[T]{
		Table: make([]*Entry[T], initSize),
		Size:  initSize,
	}
}

func (h *HashTable[T]) Hash(key string) int {
	hasher := fnv.New32a()
	hasher.Write([]byte(key))
	return int(hasher.Sum32()) % h.Size
}

func (h *HashTable[T]) Set(key string, value T) {
	// Check if we need to resize the hash table
	if float64(h.Count)/float64(h.Size) > loadFactor {
		h.resize()
	}

	index := h.Hash(key)
	entry := &Entry[T]{Key: key, Value: value, Next: nil}
	if h.Table[index] == nil {
		h.Table[index] = entry
		h.Count++
	} else {
		curr := h.Table[index]
		for curr != nil {
			if curr.Key == key {
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

func (h *HashTable[T]) resize() {
	newSize := h.Size * 2
	newTable := make([]*Entry[T], newSize)
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

func (h *HashTable[T]) Delete(key string) bool {
	index := h.Hash(key)
	if h.Table[index] == nil {
		// The key doesn't exist
		return false
	}

	// Special case: check if the key matches the first entry in the list
	if h.Table[index].Key == key {
		h.Table[index] = h.Table[index].Next
		return true
	}

	prev := h.Table[index]
	curr := prev.Next
	for curr != nil {
		if curr.Key == key {
			prev.Next = curr.Next // Bypass the entry to be deleted
			return true
		}
		prev = curr
		curr = curr.Next
	}

	// If we reach here, the key wasn't found in the list
	return false
}

func (h *HashTable[T]) Get(key string) (T, bool) {
	index := h.Hash(key)
	curr := h.Table[index]
	for curr != nil {
		if curr.Key == key {
			return curr.Value, true
		}
		curr = curr.Next
	}
	var zero T
	return zero, false
}

// Len returns the number of elements in the hash table
func (h *HashTable[T]) Len() int {
	return h.Count
}

// Empty returns true if the hash table is empty
func (h *HashTable[T]) Empty() bool {
	return h.Count == 0
}
