package db

import (
	"bytes"
	"fmt"
	"sort"
)

// https://en.wikipedia.org/wiki/Radix_tree

type edges[T any] []Edge[T]

func (e edges[T]) Len() int {
	return len(e)
}

func (e edges[T]) Less(i, j int) bool {
	return bytes.Compare(e[i].label, e[j].label) < 0
}

func (e edges[T]) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

func (e edges[T]) Sort() {
	sort.Sort(e)
}

type LeafNode[T any] struct {
	key []byte
	val T
}

type Edge[T any] struct {
	label []byte
	next  *Node[T]
}

type Node[T any] struct {
	prefix []byte
	edges  edges[T]
	leaf   *LeafNode[T] // nil if not a leaf
}

func (n *Node[T]) isLeaf() bool {
	return n.leaf != nil
}

// addEdgeByIdx inserts the given edge into the node's edges in sorted order.
func (n *Node[T]) addEdgeByIdx(idx int, e Edge[T]) {
	num := len(n.edges)

	if idx == num {
		n.edges = append(n.edges, e)
	} else {
		n.edges = append(n.edges, Edge[T]{})
		copy(n.edges[idx+1:], n.edges[idx:num])
		n.edges[idx] = e
	}
}

// addEdge, find the closest index where the edge should be inserted.
// and the longest common prefix between the label and the edge label at the closest index.
func (n *Node[T]) addEdge(e Edge[T]) {
	idx := n.findClosestPrefixIndex(e.label)
	n.addEdgeByIdx(idx, e)
}

func (n *Node[T]) findClosestPrefixIndex(label []byte) int {
	left, right := 0, len(n.edges)-1
	closestIndex := -1 // Initialize with -1 to indicate "not found"

	for left <= right {
		mid := (left + right) / 2
		cmp := bytes.Compare(n.edges[mid].label, label)

		if cmp == 0 || bytes.HasPrefix(label, n.edges[mid].label) || bytes.HasPrefix(n.edges[mid].label, label) {
			return mid
		}

		if cmp > 0 {
			right = mid - 1
		} else {
			closestIndex = mid
			left = mid + 1
		}
	}

	return closestIndex + 1
}

func (n *Node[T]) updateEdge(idx int, node *Node[T]) {
	num := len(n.edges)

	if idx < num {
		n.edges[idx].next = node
		n.edges[idx].label = node.prefix
	} else {
		panic("edge not found")
	}
}

// walkEdge returns the next node if the edge with the given label exists.
// if the edge with the given label does not exist, it returns the closest index where the edge should be inserted.
// and the longest common prefix between the label and the edge label at the closest index.
func (n *Node[T]) walkEdge(label []byte) (*Node[T], int, []byte) {
	left, right := 0, len(n.edges)-1
	closestIndex := -1 // Initialize with -1 to indicate "not found"
	for left <= right {
		mid := (left + right) / 2
		cmp := bytes.Compare(n.edges[mid].label, label)

		matchBytes := longestCommonPrefix(n.edges[mid].label, label)
		if cmp == 0 || len(matchBytes) > 0 {
			return n.edges[mid].next, mid, matchBytes
		}

		if cmp > 0 {
			right = mid - 1
		} else {
			closestIndex = mid
			left = mid + 1
		}
	}

	return nil, closestIndex + 1, nil
}

func (n *Node[T]) removeEdgeByIdx(idx int) {

	num := len(n.edges)
	if idx < num {
		n.edges = append(n.edges[:idx], n.edges[idx+1:]...)
	} else {
		panic("edge not found")
	}
}

type RaxTree[T any] struct {
	root *Node[T]
	size int
}

func NewRaxTree[T any]() *RaxTree[T] {
	return &RaxTree[T]{root: &Node[T]{prefix: []byte{}}}
}

func (t *RaxTree[T]) Len() int {
	return t.size
}

func (t *RaxTree[T]) Insert(key []byte, val T) bool {
	var parent *Node[T]
	n := t.root
	search := key
	for {
		if len(search) == 0 {
			if n.isLeaf() {
				n.leaf.val = val
				return false
			}

			n.leaf = &LeafNode[T]{
				val: val,
				key: key,
			}
			t.size++
			return true
		}

		parent = n
		n, idx, matchBytes := n.walkEdge(search)

		if n == nil {
			e := Edge[T]{label: search, next: &Node[T]{prefix: search, leaf: &LeafNode[T]{key: search, val: val}}}
			parent.addEdgeByIdx(idx, e)
			t.size++
			return true
		}

		if len(matchBytes) == len(n.prefix) {
			search = search[len(matchBytes):]
			continue
		}

		// Split the node at the common prefix
		splitNode := &Node[T]{prefix: matchBytes}
		parent.updateEdge(idx, splitNode)

		// The remaining part of the existing node after the split
		n.prefix = n.prefix[len(matchBytes):]
		e := Edge[T]{label: n.prefix, next: n}
		splitNode.addEdge(e)

		search = search[len(matchBytes):]
		if len(search) == 0 {
			splitNode.leaf = &LeafNode[T]{key: key, val: val}
			t.size++
			return true
		}

		e = Edge[T]{label: search, next: &Node[T]{prefix: search, leaf: &LeafNode[T]{key: key, val: val}}}
		splitNode.addEdge(e)

		t.size++
		return true
	}
}

func (t *RaxTree[T]) Find(key []byte) (T, bool) {
	n := t.root
	search := key

	for len(search) > 0 {
		next, _, matchBytes := n.walkEdge(search)
		if next == nil {
			var zero T
			return zero, false
		}
		search = search[len(matchBytes):]

		n = next
		if len(search) == 0 {
			break
		}
	}

	if n.isLeaf() {
		return n.leaf.val, true
	}

	var zero T
	return zero, false
}

func (t *RaxTree[T]) Delete(key []byte) bool {
	var (
		parent, next *Node[T] // To keep track of the parent node
		idx          int
		matchBytes   []byte
	)
	n := t.root
	search := key

	for len(search) > 0 {
		next, idx, matchBytes = n.walkEdge(search)
		if next == nil {
			// Key not found in the tree
			return false
		}

		if bytes.Equal(next.prefix, search) {
			// The search key matches the next node's prefix
			parent = n
			n = next
			break
		}

		if bytes.HasPrefix(next.prefix, matchBytes) {
			// The search key matches the next node's prefix
			search = search[len(matchBytes):]
			parent = n
			n = next
			if len(search) == 0 {
				break
			}
		} else {
			// Partial match, so the key is not in the tree
			return false
		}
	}

	if !n.isLeaf() {
		// Key doesn't exist in the tree
		return false
	}

	// Remove the leaf node
	parent.removeEdgeByIdx(idx)

	// Check if the parent node needs to be updated or removed
	for len(parent.edges) == 1 && !parent.isLeaf() {
		// The parent node has only one edge and is not a leaf
		// Merge it with its child node and update the parent
		child := parent.edges[0].next
		parent.prefix = append(parent.prefix, child.prefix...)
		parent.edges = child.edges
		parent.leaf = child.leaf
	}

	t.size--
	return true
}

func (t *RaxTree[T]) Print() {
	printRaxTree(t.root, "")
}

// Your existing printRaxTree function or make it an unexported function if it's part of the same package
func printRaxTree[T any](node *Node[T], prefix string) {
	if node.isLeaf() {
		fmt.Printf("%s[Leaf: Key = %s, Value = %v, Prefix = %s]\n", prefix, node.leaf.key, node.leaf.val, node.prefix)
		return
	}

	fmt.Printf("%s[Node: Prefix = %s]\n", prefix, node.prefix)

	for _, edge := range node.edges {
		fmt.Printf("%s|--Edge: Label = %s\n", prefix, edge.label)
		printRaxTree(edge.next, prefix+"  ")
	}
}

func longestCommonPrefix(a, b []byte) []byte {
	var i int
	for i = 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			break
		}
	}
	return a[:i]
}
