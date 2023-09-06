package db

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRaxTreeInsertAndFind(t *testing.T) {
	assert := assert.New(t)
	tree := NewRaxTree[string]()

	tree.Insert([]byte("apple"), "fruit")
	val, found := tree.Find([]byte("apple"))

	assert.True(found)
	assert.Equal("fruit", val)

	_, found = tree.Find([]byte("orange"))
	assert.False(found)

	tree.Insert([]byte("appetizer"), "food")

	tree.Print()
	val, found = tree.Find([]byte("appetizer"))
	assert.True(found)
	assert.Equal("food", val)

	val, found = tree.Find([]byte("apple"))
	assert.True(found)
	assert.Equal("fruit", val)

	tree.Insert([]byte("banana"), "fruit")
	val, found = tree.Find([]byte("banana"))
	assert.True(found)
	assert.Equal("fruit", val)

	assert.Equal(3, tree.Len())
}

func TestRaxTreeDelete(t *testing.T) {
	assert := assert.New(t)
	tree := NewRaxTree[string]()

	tree.Insert([]byte("apple"), "fruit")
	tree.Insert([]byte("appetizer"), "food")

	// Test deleting an existing key
	deleted := tree.Delete([]byte("apple"))
	assert.True(deleted)
	_, found := tree.Find([]byte("apple"))
	assert.False(found)

	// Test deleting a non-existing key
	deleted = tree.Delete([]byte("orange"))
	assert.False(deleted)

	// Test deleting an existing key
	deleted = tree.Delete([]byte("appetizer"))
	assert.True(deleted)
	_, found = tree.Find([]byte("appetizer"))
	assert.False(found)

	// Test deleting the same key again
	deleted = tree.Delete([]byte("appetizer"))
	assert.False(deleted)
}

func TestRaxTree(t *testing.T) {
	tree := NewRaxTree[int]()

	// Insertions
	assert.True(t, tree.Insert([]byte("apple"), 1))
	assert.True(t, tree.Insert([]byte("apricot"), 2))
	assert.True(t, tree.Insert([]byte("banana"), 3))
	assert.True(t, tree.Insert([]byte("batman"), 4))
	assert.True(t, tree.Insert([]byte("bat"), 5))
	assert.True(t, tree.Insert([]byte("appetite"), 6))

	// Printing the tree after insertions
	fmt.Println("Tree after insertions:")
	tree.Print()

	// Deletions
	fmt.Println("Deleting 'apricot'...")
	assert.True(t, tree.Delete([]byte("apricot")))
	fmt.Println("Deleting 'bat'...")
	assert.True(t, tree.Delete([]byte("bat")))
	fmt.Println("Deleting 'nonexistent'...")
	assert.False(t, tree.Delete([]byte("nonexistent")))

	// Printing the tree after deletions
	fmt.Println("Tree after deletions:")
	tree.Print()

	// Complex Insertions
	fmt.Println("Inserting 'apex'...")
	assert.True(t, tree.Insert([]byte("apex"), 7))
	fmt.Println("Inserting 'appetizer'...")
	assert.True(t, tree.Insert([]byte("appetizer"), 8))

	// Printing the tree after complex insertions
	fmt.Println("Tree after complex insertions:")
	tree.Print()

	// Complex Deletions
	fmt.Println("Deleting 'apple'...")
	assert.True(t, tree.Delete([]byte("apple")))
	fmt.Println("Deleting 'banana'...")
	assert.True(t, tree.Delete([]byte("banana")))

	// Printing the tree after complex deletions
	fmt.Println("Tree after complex deletions:")
	tree.Print()
}
