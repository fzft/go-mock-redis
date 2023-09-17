package hredis

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRedisFormatCommand(t *testing.T) {
	// Test case 1: With string and integer
	argv1, err1 := redisFormatCommand("Hello %s %d", "world", 2023)
	assert.Nil(t, err1, "Expected no error")
	assert.Equal(t, []string{"Hello", "world", "2023"}, argv1, "Array mismatch")

	// Test case 2: Without any format specifier
	argv2, err2 := redisFormatCommand("Hello world", "extra", 42)
	assert.Nil(t, err2, "Expected no error")
	assert.Equal(t, []string{"Hello", "world"}, argv2, "Array mismatch")

	// Test case 3: With unsupported format specifier
	_, err3 := redisFormatCommand("Hello %c world", 'A')
	assert.Equal(t, "Unsupported format specifier: c", err3.Error(), "Expected an error indicating unsupported format specifier")

	// Test case 4: With not enough arguments
	_, err4 := redisFormatCommand("Hello %s %d", "world")
	assert.Equal(t, "Not enough arguments", err4.Error(), "Expected an error indicating not enough arguments")
}
