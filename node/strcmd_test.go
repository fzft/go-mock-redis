package node

import (
	"bytes"
	"github.com/fzft/go-mock-redis/db"
	"github.com/stretchr/testify/assert"
	"testing"
)

type TestConn struct {
	Buffer bytes.Buffer // buffer to capture output
}

func (t *TestConn) Read() ([]byte, error) {
	// For testing, you might want to mock this input.
	return nil, nil
}

func (t *TestConn) Write(b []byte) error {
	_, err := t.Buffer.Write(b) // Capture output to buffer
	// For testing, you might want to capture this output and validate it.
	return err
}

func (t *TestConn) Ip() string {
	return ""
}

func (t *TestConn) Close() error {
	return nil
}

func (t *TestConn) Fd() int {
	return 0
}

func TestSetAndGetCommand(t *testing.T) {
	// Initialize client
	testDb := db.New(0)
	client := NewClient(1, 0, &TestConn{}, 2, testDb)

	// Test SET command
	cmd := NewStrCmd(client, testDb)
	cmd.c.argc = 4
	cmd.c.argv = []*db.RedisObj{
		&db.RedisObj{Value: "SET"},
		&db.RedisObj{Value: "mykey"},
		&db.RedisObj{Value: "myvalue"},
		&db.RedisObj{Value: "NX"},
	}
	cmd.Set()

	myValue, exist := testDb.LookupKeyRead("mykey")

	assert.Equal(t, true, exist)

	// Check if key-value pair is set in db
	assert.Equal(t, &db.RedisObj{Value: "myvalue", LRU: 100}, myValue)

	// Test GET command
	cmd.c.argc = 2
	cmd.c.argv = []*db.RedisObj{
		&db.RedisObj{Value: "GET"},
		&db.RedisObj{Value: "mykey"},
	}

	cmd.Get()

}
