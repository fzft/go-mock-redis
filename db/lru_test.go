package db

import (
	"fmt"
	"testing"
)

func TestEvictionPoolPopulate(t *testing.T) {
	// Setup
	lru := NewLRU()
	sampleDict := NewHashTable[string, *RedisObj](10)
	keyDict := NewHashTable[string, *RedisObj](10)

	obj1 := &RedisObj{Key: "key1", Value: "val1", LRU: 5}
	obj2 := &RedisObj{Key: "key2", Value: "val2", LRU: 3}
	obj3 := &RedisObj{Key: "key3", Value: "val3", LRU: 7}
	obj4 := &RedisObj{Key: "key4", Value: "val4", LRU: 2}

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
	//assert.Equal(t, "key4", lru.ep[0].Key, "Expected key4 as least recently used")
	//assert.Equal(t, "key2", lru.ep[1].Key, "Expected key2 as next least recently used")
	//assert.Equal(t, "key1", lru.ep[2].Key, "Expected key1 as next least recently used")
	//assert.Equal(t, "key3", lru.ep[3].Key, "Expected key3 as next least recently used")
	//

	for _, ev := range lru.ep {
		fmt.Printf("idle: %d, key: %s\n", ev.Idle, ev.Key)
	}
}
