package xray

import (
	"fmt"
)

// HashedSet implements a set with order-independent hash computation.
// Uses dual DJB2 hashing with XOR accumulator for O(1) hash updates.
// Compatible with @remnawave/hashed-set algorithm.
type HashedSet struct {
	items    map[string]struct{}
	hashHigh uint32
	hashLow  uint32
}

// NewHashedSet creates a new empty HashedSet.
func NewHashedSet() *HashedSet {
	return &HashedSet{
		items:    make(map[string]struct{}),
		hashHigh: 0,
		hashLow:  0,
	}
}

// djb2Dual computes dual DJB2 hash values for a string.
// Uses seeds 5381 (high) and 5387 (low) with different mixing functions.
// Algorithm matches @remnawave/hashed-set exactly.
func djb2Dual(str string) (high, low uint32) {
	var h int32 = 5381
	var l int32 = 5387

	for i := 0; i < len(str); i++ {
		c := int32(str[i])
		h = ((h << 5) + h + c)    // h = h * 33 + char
		l = ((l << 6) + l + c*37) // l = l * 65 + char * 37
	}

	return uint32(h), uint32(l)
}

// Add adds a string to the set. If already present, does nothing.
func (s *HashedSet) Add(str string) {
	if _, exists := s.items[str]; !exists {
		s.items[str] = struct{}{}
		high, low := djb2Dual(str)
		s.hashHigh ^= high
		s.hashLow ^= low
	}
}

// Delete removes a string from the set. If not present, does nothing.
func (s *HashedSet) Delete(str string) {
	if _, exists := s.items[str]; exists {
		delete(s.items, str)
		high, low := djb2Dual(str)
		// XOR is self-inverse: XOR same value removes it
		s.hashHigh ^= high
		s.hashLow ^= low
	}
}

// Has returns true if the string is in the set.
func (s *HashedSet) Has(str string) bool {
	_, exists := s.items[str]
	return exists
}

// Size returns the number of items in the set.
func (s *HashedSet) Size() int {
	return len(s.items)
}

// Clear removes all items from the set.
func (s *HashedSet) Clear() {
	s.items = make(map[string]struct{})
	s.hashHigh = 0
	s.hashLow = 0
}

// Hash64String returns the 16-character lowercase hex string representation
// of the set's hash. Format: 8 chars high + 8 chars low.
// Empty set returns "0000000000000000".
func (s *HashedSet) Hash64String() string {
	return fmt.Sprintf("%08x%08x", s.hashHigh, s.hashLow)
}

// Items returns a copy of all items in the set.
func (s *HashedSet) Items() []string {
	result := make([]string, 0, len(s.items))
	for item := range s.items {
		result = append(result, item)
	}
	return result
}
