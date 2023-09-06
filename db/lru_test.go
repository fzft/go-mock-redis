package db

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEvictionPoolPopulate(t *testing.T) {
	// Setup
	lru := NewLRU(New(0), 0, 100)
	sampleDict := NewHashTable[string, *RedisObj](10)
	keyDict := NewHashTable[string, *RedisObj](10)

	obj1 := &RedisObj{Type: StringType, Value: "val1", LRU: 5}
	obj2 := &RedisObj{Type: StringType, Value: "val2", LRU: 3}
	obj3 := &RedisObj{Type: StringType, Value: "val3", LRU: 7}
	obj4 := &RedisObj{Type: StringType, Value: "val4", LRU: 2}

	keyDict.Set("key1", obj1)
	keyDict.Set("key2", obj2)
	keyDict.Set("key3", obj3)
	keyDict.Set("key4", obj4)

	// The sample dictionary should only get keys, not values
	sampleDict.Set("key1", nil)
	sampleDict.Set("key2", nil)
	sampleDict.Set("key3", nil)
	sampleDict.Set("key4", nil)

	lru.EvictionPoolPopulate(sampleDict, keyDict)

	// Using assert to validate the results
	assert.Equal(t, "key3", lru.ep[0].Key, "Expected key3 as least recently used")
	assert.Equal(t, "key1", lru.ep[1].Key, "Expected key1 as next least recently used")
	assert.Equal(t, "key2", lru.ep[2].Key, "Expected key2 as next least recently used")
	assert.Equal(t, "key4", lru.ep[3].Key, "Expected key4 as next least recently used")

}

func TestEvictionPoolReplaceMiddleSlot(t *testing.T) {
	// Setup
	lru := NewLRU(New(0), 0, 100)
	sampleDict := NewHashTable[string, *RedisObj](10)
	keyDict := NewHashTable[string, *RedisObj](10)

	// Initially fill the eviction pool
	obj1 := &RedisObj{Type: StringType, Value: "val1", LRU: 5}
	obj2 := &RedisObj{Type: StringType, Value: "val2", LRU: 3}
	obj3 := &RedisObj{Type: StringType, Value: "val3", LRU: 7}
	obj4 := &RedisObj{Type: StringType, Value: "val4", LRU: 2}

	keyDict.Set("key1", obj1)
	keyDict.Set("key2", obj2)
	keyDict.Set("key3", obj3)
	keyDict.Set("key4", obj4)

	sampleDict.Set("key1", nil)
	sampleDict.Set("key2", nil)
	sampleDict.Set("key3", nil)
	sampleDict.Set("key4", nil)

	// Add a new key with an LRU that would be inserted in the middle of the eviction pool
	obj5 := &RedisObj{Type: StringType, Value: "val5", LRU: 6}
	keyDict.Set("key5", obj5)
	sampleDict.Set("key5", nil)

	lru.EvictionPoolPopulate(sampleDict, keyDict)

	//// Using assert to validate the results
	assert.Equal(t, "key5", lru.ep[0].Key, "Expected key5 as the most recently used")
	assert.Equal(t, "key1", lru.ep[1].Key, "Expected key1 as the next most recently used") // This is the key that should have been inserted in the middle
	assert.Equal(t, "key2", lru.ep[2].Key, "Expected key2 as next")
	assert.Equal(t, "key4", lru.ep[3].Key, "Expected key4 as the least recently used")
}
