package db

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewList(t *testing.T) {
	list := NewList[int]()
	assert.Nil(t, list.Head)
	assert.Nil(t, list.Tail)
	assert.Equal(t, 0, list.Len())
}

func TestAddNodeHead(t *testing.T) {
	list := NewList[int]()
	list.AddNodeHead(5)
	assert.Equal(t, 5, list.Head.Value)
	assert.Equal(t, 5, list.Tail.Value)
	assert.Equal(t, 1, list.Len())
}

func TestAddNodeTail(t *testing.T) {
	list := NewList[int]()
	list.AddNodeTail(5)
	assert.Equal(t, 5, list.Head.Value)
	assert.Equal(t, 5, list.Tail.Value)
	assert.Equal(t, 1, list.Len())

	list.AddNodeTail(10)
	assert.Equal(t, 5, list.Head.Value)
	assert.Equal(t, 10, list.Tail.Value)
	assert.Equal(t, 2, list.Len())
}

func TestEmpty(t *testing.T) {
	list := NewList[int]()
	list.AddNodeTail(5)
	list.AddNodeTail(10)
	list.Empty()
	assert.Nil(t, list.Head)
	assert.Nil(t, list.Tail)
	assert.Equal(t, 0, list.Len())
}

func TestRelease(t *testing.T) {
	list := NewList[int]()
	list.AddNodeTail(5)
	list.AddNodeTail(10)
	list.Release()
	assert.Nil(t, list.Head)
	assert.Nil(t, list.Tail)
	assert.Equal(t, 0, list.Len())
}

func TestInsertNode(t *testing.T) {
	list := NewList[int]()
	list.AddNodeTail(5)
	list.AddNodeTail(10)
	err := list.InsertNode(list.Head, 7, true)
	assert.Nil(t, err)
	assert.Equal(t, 3, list.Len())
	assert.Equal(t, 5, list.Head.Value)
	assert.Equal(t, 10, list.Tail.Value)
	assert.Equal(t, 7, list.Head.Next.Value)
}

func TestRemoveNode(t *testing.T) {
	list := NewList[int]()
	list.AddNodeTail(5)
	list.AddNodeTail(10)
	err := list.RemoveNode(list.Head)
	assert.Nil(t, err)
	assert.Equal(t, 1, list.Len())
	assert.Equal(t, 10, list.Head.Value)
	assert.Equal(t, 10, list.Tail.Value)
}
