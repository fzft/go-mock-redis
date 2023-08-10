package db

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestHashTableSetAndGet(t *testing.T) {
	ht := NewHashTable[int](10)
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
	ht := NewHashTable[int](10)
	ht.Set("one", 1)
	ht.Delete("one")

	_, exists := ht.Get("one")
	assert.False(t, exists, "Expected key 'one' to be deleted")
}

func TestHashTableResize(t *testing.T) {
	ht := NewHashTable[int](10)

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
