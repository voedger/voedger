/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package set

import (
	"encoding/binary"
	"fmt"
	"iter"
	"math/bits"
	"strings"
)

// Set represents a set of values of type V.
//
// V must be uint8.
type Set[V ~uint8] struct {
	bitmap   [4]uint64 // bit set flag
	readOnly bool
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

// Makes new read only Set from specified values.
func FromRO[V ~uint8](values ...V) Set[V] {
	var s Set[V]
	s.Set(values...)
	s.SetReadOnly()
	return s
}

// Makes new Set from specified iterator.
func Collect[V ~uint8](it iter.Seq[V]) Set[V] {
	var s Set[V]
	s.Collect(it)
	return s
}

// All returns iterator which calls visit for each value in Set.
func (s Set[V]) All() iter.Seq2[int, V] {
	return func(visit func(int, V) bool) {
		idx := 0
		for i, b := range s.bitmap {
			if b == 0 {
				continue
			}
			l := bits.TrailingZeros64(b)
			h := uintSize - bits.LeadingZeros64(b)
			for v := l; v < h; v++ {
				if b&(1<<v) != 0 {
					if !visit(idx, V(i*uintSize+v)) {
						return
					}
					idx++
				}
			}
		}
	}
}

// Represents Set as array.
//
// If Set is empty, returns nil.
func (s Set[V]) AsArray() (a []V) {
	for v := range s.Values() {
		a = append(a, v)
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

// Backward calls visit for each value in Set in backward order.
func (s Set[V]) Backward() iter.Seq[V] {
	return func(visit func(V) bool) {
		for i := len(s.bitmap) - 1; i >= 0; i-- {
			b := s.bitmap[i]
			if b == 0 {
				continue
			}
			h := uintSize - bits.LeadingZeros64(b)
			l := bits.TrailingZeros64(b)
			for v := h - 1; v >= l; v-- {
				if b&(1<<v) != 0 {
					if !visit(V(i*uintSize + v)) {
						return
					}
				}
			}
		}
	}
}

// Chunk returns an iterator over consecutive sub-sets of up to n elements of s.
// All but the last sub-set will have length n.
// If s is empty, the sequence is empty: there is no empty sets in the sequence.
//
// # Panics:
//   - if n is less than 1.
func (s Set[V]) Chunk(n int) iter.Seq[Set[V]] {
	if n < 1 {
		panic("chunk size should be positive")
	}
	return func(visit func(Set[V]) bool) {
		chunk := Empty[V]()
		for v := range s.Values() {
			chunk.setBit(v)
			if chunk.Len() == n {
				if !visit(chunk) {
					return
				}
				chunk.ClearAll()
			}
		}
		if chunk.Len() > 0 {
			visit(chunk)
		}
	}
}

// Clears specified elements from set.
func (s *Set[V]) Clear(values ...V) {
	s.checkRO()
	for _, v := range values {
		s.bitmap[v/uintSize] &^= 1 << (v % uintSize)
	}
}

// Clears all elements from Set.
func (s *Set[V]) ClearAll() {
	s.checkRO()
	for i := range s.bitmap {
		s.bitmap[i] = 0
	}
}

// Clone returns a copy of the Set.
func (s Set[V]) Clone() Set[V] {
	return s
}

// Collect collects values from iterator into Set.
func (s *Set[V]) Collect(it iter.Seq[V]) {
	s.checkRO()
	for v := range it {
		s.setBit(v)
	}
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

// Returns is Set filled and first value set.
// If Set is empty, returns false and zero value.
func (s Set[V]) First() (V, bool) {
	for v := range s.Values() {
		return v, true
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
	s.checkRO()
	for _, v := range values {
		s.setBit(v)
	}
}

// Sets range of value to Set. Inclusive start, exclusive end.
func (s *Set[V]) SetRange(start, end V) {
	s.checkRO()
	for k := start; k < end; k++ {
		s.setBit(k)
	}
}

// Mark set readonly. This operation is irreversible.
// Useful to protect set from modification, then set used as immutable constant.
func (s *Set[V]) SetReadOnly() {
	s.readOnly = true
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

	for v := range s.Values() {
		ss = append(ss, say(v))
	}

	return fmt.Sprintf("[%v]", strings.Join(ss, " "))
}

// Values returns iterator which calls visit for each value in Set.
func (s Set[V]) Values() iter.Seq[V] {
	return func(visit func(V) bool) {
		for _, v := range s.All() {
			if !visit(v) {
				return
			}
		}
	}
}

func (s Set[V]) checkRO() {
	if s.readOnly {
		panic("set is read-only")
	}
}

func (s *Set[V]) setBit(v V) {
	s.bitmap[v/uintSize] |= 1 << (v % uintSize)
}

const uintSize = bits.UintSize
