package db

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"sync/atomic"
	"testing"
)

func TestHashTableSetAndGet(t *testing.T) {
	ht := NewHashTable[string, int](10)
	ht.Set("one", 1)
	ht.Set("two", 2)

	value, exists := ht.Get("one")
	assert.True(t, exists, "Key 'one' should exist")
	assert.Equal(t, 1, value, "Value for key 'one' should be 1")

	value, exists = ht.Get("two")
	assert.True(t, exists, "Key 'two' should exist")
	assert.Equal(t, 2, value, "Value for key 'two' should be 2")

	_, exists = ht.Get("three")
	assert.False(t, exists, "Key 'three' should not exist")
}

func TestHashTableDelete(t *testing.T) {
	ht := NewHashTable[string, int](10)
	ht.Set("one", 1)
	ht.Delete("one")

	_, exists := ht.Get("one")
	assert.False(t, exists, "Expected key 'one' to be deleted")
}

func TestHashTableResize(t *testing.T) {
	ht := NewHashTable[string, int](10)

	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key%d", i)
		ht.Set(key, i)
	}

	value, exists := ht.Get("key50")
	assert.True(t, exists, "Key 'key50' should exist")
	assert.Equal(t, 50, value, "Value for key 'key50' should be 50")

	value, exists = ht.Get("key99")
	assert.True(t, exists, "Key 'key99' should exist")
	assert.Equal(t, 99, value, "Value for key 'key99' should be 99")
}

func TestGetSomeKeys(t *testing.T) {
	table := NewHashTable[string, string](10)
	table.Set("name1", "John")
	table.Set("name2", "Doe")
	table.Set("name3", "Smith")

	keys := table.GetSomeKeys(2)
	assert.NotNil(t, keys)
	assert.Equal(t, 2, len(keys))

	for _, key := range keys {
		_, exists := table.Get(key)
		assert.True(t, exists, "Expected key to exist in table:", key)
	}
}

func TestGetSomeKeysWhenEmpty(t *testing.T) {
	table := NewHashTable[string, string](10)
	keys := table.GetSomeKeys(10)
	assert.Nil(t, keys)
}

func TestGetSomeKeysWhenCountIsGreaterThanSize(t *testing.T) {
	table := NewHashTable[string, string](10)
	table.Set("name1", "John")

	keys := table.GetSomeKeys(5)
	assert.NotNil(t, keys)
	assert.Equal(t, 1, len(keys))
	assert.Equal(t, "name1", keys[0])
}

func TestMemoryUsage(t *testing.T) {
	// Reset usedMemory for testing
	atomic.StoreInt64(&usedMemory, 0)

	// Create a new hash table
	table := NewHashTable[int, string](10)

	// Set key-value pairs
	table.Set(1, "one")
	table.Set(2, "two")
	table.Set(3, "three")

	// Calculate expected memory after inserts
	expectedMemoryAfterInserts := estimateMemoryUsage(1) + estimateMemoryUsage("one") + estimateMemoryUsage(2) + estimateMemoryUsage("two") + estimateMemoryUsage(3) + estimateMemoryUsage("three")

	// Assert memory usage after inserts
	assert.Equal(t, expectedMemoryAfterInserts, getUsedMemory(), "Memory usage after inserts is incorrect")

	// Delete a key-value pair
	table.Delete(1)

	// Calculate expected memory after delete
	expectedMemoryAfterDelete := expectedMemoryAfterInserts - estimateMemoryUsage(1) - estimateMemoryUsage("one")

	// Assert memory usage after delete
	assert.Equal(t, expectedMemoryAfterDelete, getUsedMemory(), "Memory usage after delete is incorrect")
}
