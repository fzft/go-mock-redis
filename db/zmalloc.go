package db

import (
	"sync/atomic"
	"unsafe"
)

const MAX_MEMORY = 1024 * 1024 * 1024

var usedMemory int64 = 0

// IncreaseUsedMemory increases the used memory counter
func IncreaseUsedMemory(v any) {
	n := estimateMemoryUsage(v)
	updateZmallocStatAlloc(n)
}

// DecreaseUsedMemory decreases the used memory counter
func DecreaseUsedMemory(v any) {
	n := estimateMemoryUsage(v)
	updateZmallocStatFree(n)
}

func updateZmallocStatAlloc(n int64) {
	atomic.AddInt64(&usedMemory, n)
}

func updateZmallocStatFree(n int64) {
	atomic.AddInt64(&usedMemory, -n)
}

func getUsedMemory() int64 {
	return atomic.LoadInt64(&usedMemory)
}

func estimateMemoryUsage(v any) int64 {
	switch value := v.(type) {
	case int:
		return int64(unsafe.Sizeof(value))
	case float64:
		return int64(unsafe.Sizeof(value))
	case string:
		// 16 bytes for string header on 64-bit system + actual string content
		return int64(16 + len(value))
	case []int:
		// 24 bytes for slice header on 64-bit system + content
		return int64(24 + len(value)*int(unsafe.Sizeof(value[0])))
	// ... add more types as needed
	default:
		return 0
	}
}
