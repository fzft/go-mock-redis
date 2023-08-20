package db

type ListNode[T any] struct {
	Prev  *ListNode[T]
	Next  *ListNode[T]
	Value T
}

type ListIter[T any] struct {
	Next      *ListNode[T]
	Direction int
}

type List[T any] struct {
	Head   *ListNode[T]
	Tail   *ListNode[T]
	Length int
}

const (
	DIRECTION_HEAD = iota
	DIRECTION_TAIL
)

func NewList[T any]() *List[T] {
	return &List[T]{}
}

// Empty the list
func (l *List[T]) Empty() {
	current := l.Head
	for current != nil {
		next := current.Next
		current = next
	}
	l.Head, l.Tail = nil, nil
	l.Length = 0
}

// Release the list
func (l *List[T]) Release() {
	l.Empty()
}

func (l *List[T]) AddNodeHead(value T) {
	node := &ListNode[T]{Value: value}
	if l.Head == nil {
		l.Head, l.Tail = node, node
	} else {
		node.Next, l.Head.Prev, l.Head = l.Head, node, node
	}
	l.Length++
}

func (l *List[T]) AddNodeTail(value T) {
	node := &ListNode[T]{Value: value}
	if l.Tail == nil {
		l.Head, l.Tail = node, node
	} else {
		node.Prev, l.Tail.Next, l.Tail = l.Tail, node, node
	}
	l.Length++
}

func (l *List[T]) InsertNode(oldNode *ListNode[T], value T, after bool) error {
	node := &ListNode[T]{Value: value}
	if after {
		if oldNode.Next == nil {
			l.AddNodeTail(value)
			return nil
		}
		node.Prev = oldNode
		node.Next = oldNode.Next
		oldNode.Next = node
		if node.Next != nil { // Ensure it's not the last node
			node.Next.Prev = node
		}
	} else {
		if oldNode.Prev == nil {
			l.AddNodeHead(value)
			return nil
		}
		node.Prev, node.Next, oldNode.Prev, node.Prev.Next = oldNode.Prev, oldNode, node, node
	}
	l.Length++
	return nil
}

// RemoveNode a node from the list
func (l *List[T]) RemoveNode(node *ListNode[T]) error {
	if node.Prev != nil {
		node.Prev.Next = node.Next
	} else {
		l.Head = node.Next
	}
	if node.Next != nil {
		node.Next.Prev = node.Prev
	} else {
		l.Tail = node.Prev
	}
	node.Next, node.Prev = nil, nil
	l.Length--
	return nil
}

// Len ...
func (l *List[T]) Len() int {
	return l.Length
}
