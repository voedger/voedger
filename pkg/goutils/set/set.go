/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package set

import (
	"encoding/binary"
	"fmt"
	"math/bits"
	"strings"
)

// Set represents a set of values of type V.
//
// V must be uint8.
type Set[V ~uint8] struct {
	bitmap [4]uint64 // bit set flag
}

// Makes new empty Set of specified value type. Same as `Set[V]{}`.
func Empty[V ~uint8]() Set[V] {
	return Set[V]{}
}

// Makes new Set from specified values.
func From[V ~uint8](values ...V) Set[V] {
	var s Set[V]
	s.Set(values...)
	return s
}

// Represents Set as array.
//
// If Set is empty, returns nil.
func (s Set[V]) AsArray() (a []V) {
	for i, b := range s.bitmap {
		if b == 0 {
			continue
		}
		l := bits.TrailingZeros64(b)
		h := uintSize - bits.LeadingZeros64(b)
		for v := l; v < h; v++ {
			if b&(1<<v) != 0 {
				a = append(a, V(i*uintSize+v))
			}
		}
	}
	return a
}

// Returns Set bitmap as big-endian bytes.
func (s Set[V]) AsBytes() []byte {
	const (
		size = 4 * 8 // 4 * 8 = 32
		ofs  = 24    // 4 * 8 - 8 = 24
	)
	buf := make([]byte, size)
	for i := range s.bitmap {
		binary.BigEndian.PutUint64(buf[ofs-i*8:], s.bitmap[i])
	}
	return buf
}

// Clears specified elements from set.
func (s *Set[V]) Clear(values ...V) {
	for _, v := range values {
		s.bitmap[v/uintSize] &^= 1 << (v % uintSize)
	}
}

// Clears all elements from Set.
func (s *Set[V]) ClearAll() {
	for i := range s.bitmap {
		s.bitmap[i] = 0
	}
}

// Clone returns a copy of the Set.
func (s Set[V]) Clone() Set[V] {
	return s
}

// Returns is Set contains specified value.
func (s Set[V]) Contains(v V) bool {
	return s.bitmap[v/uintSize]&(1<<(v%uintSize)) != 0
}

// Returns is Set contains all from specified values.
func (s Set[V]) ContainsAll(values ...V) bool {
	for _, v := range values {
		if !s.Contains(v) {
			return false
		}
	}
	return true
}

// Returns is Set contains at least one from specified values.
// If values is empty, returns true.
func (s Set[V]) ContainsAny(values ...V) bool {
	for _, v := range values {
		if s.Contains(v) {
			return true
		}
	}
	return len(values) == 0
}

// Enumerate calls visit for each value in Set.
func (s Set[V]) Enumerate(visit func(V)) {
	for i, b := range s.bitmap {
		if b == 0 {
			continue
		}
		l := bits.TrailingZeros64(b)
		h := uintSize - bits.LeadingZeros64(b)
		for v := l; v < h; v++ {
			if b&(1<<v) != 0 {
				visit(V(i*uintSize + v))
			}
		}
	}
}

// Returns is Set filled and first value set.
// If Set is empty, returns false and zero value.
func (s Set[V]) First() (V, bool) {
	for i, b := range s.bitmap {
		if b == 0 {
			continue
		}
		if l := bits.TrailingZeros64(b); l < uintSize {
			return V(i*uintSize + l), true
		}
	}

	return V(0), false
}

// Returns count of values in Set.
func (s Set[V]) Len() int {
	c := 0
	for _, b := range s.bitmap {
		c += bits.OnesCount64(b)
	}
	return c
}

// Sets specified values to Set.
func (s *Set[V]) Set(values ...V) {
	for _, v := range values {
		s.bitmap[v/uintSize] |= 1 << (v % uintSize)
	}
}

// Sets range of value to Set. Inclusive start, exclusive end.
func (s *Set[V]) SetRange(start, end V) {
	for k := start; k < end; k++ {
		s.Set(k)
	}
}

// Renders Set in human-readable form, without `ValueTypeName_` prefixes,
// suitable for debugging or error messages
func (s Set[V]) String() string {

	say := func(v any) string {
		if trimV, ok := v.(interface{ TrimString() string }); ok {
			return trimV.TrimString()
		}
		return fmt.Sprintf("%v", v)
	}

	ss := make([]string, 0, s.Len())

	for i, b := range s.bitmap {
		if b == 0 {
			continue
		}
		l := bits.TrailingZeros64(b)
		h := uintSize - bits.LeadingZeros64(b)
		for v := l; v < h; v++ {
			if b&(1<<v) != 0 {
				ss = append(ss, say(V(i*uintSize+v)))
			}
		}
	}

	return fmt.Sprintf("[%v]", strings.Join(ss, " "))
}

const uintSize = bits.UintSize
