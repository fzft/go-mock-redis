package db

type sentinel struct{}

type Set[T any] struct {
	data *HashTable[T, sentinel]
}

// NewSet creates a new Set
func NewSet[T any](initSize int) *Set[T] {
	return &Set[T]{
		data: NewHashTable[T, sentinel](initSize),
	}
}

// Add inserts a key into the set
func (s *Set[T]) Add(key T) {
	s.data.Set(key, sentinel{})
}

// Contains checks if a key is in the set
func (s *Set[T]) Contains(key T) bool {
	_, exists := s.data.Get(key)
	return exists
}

// Remove deletes a key from the set
func (s *Set[T]) Remove(key T) {
	s.data.Delete(key)
}
