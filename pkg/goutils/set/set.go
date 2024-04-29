/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package set

import (
	"fmt"
	"math/bits"
	"strings"
)

// Set represents a set of values of type V.
//
// V must be int8 or uint8.
type Set[V ~int8 | ~uint8] struct {
	uint64 // bit set flag
}

// Makes new Set from specified values.
func From[V ~int8 | ~uint8](values ...V) Set[V] {
	var s Set[V]
	s.Set(values...)
	return s
}

// Represents Set as array.
//
// If Set is empty, returns nil.
func (s Set[V]) AsArray() []V {
	if s.uint64 == 0 {
		return nil
	}
	var a []V
	for v := V(0); v < bits.UintSize; v++ {
		if s.Contains(v) {
			a = append(a, v)
		}
	}
	return a
}

// Returns Set as uint64.
func (s Set[V]) AsInt64() uint64 {
	return s.uint64
}

// Clears specified elements from set.
func (s *Set[V]) Clear(values ...V) {
	for _, v := range values {
		s.uint64 &^= 1 << v
	}
}

// Clears all elements from Set.
func (s *Set[V]) ClearAll() {
	s.uint64 = 0
}

// TODO: ClearRange

// Returns is Set contains specified value.
func (s Set[V]) Contains(v V) bool {
	return s.uint64&(1<<v) != 0
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
func (s Set[V]) First() (bool, V) {
	for v := V(0); v < bits.UintSize; v++ {
		if s.Contains(v) {
			return true, v
		}
	}
	return false, V(0)
}

// Returns count of values in Set.
func (s Set[V]) Len() int {
	return bits.OnesCount64(s.uint64)
}

// Puts uint64 value to Set.
func (s *Set[V]) PutInt64(v uint64) {
	s.uint64 = v
}

// Sets specified values to Set.
func (s *Set[V]) Set(values ...V) {
	for _, v := range values {
		s.uint64 |= 1 << v
	}
}

// Sets range of value to Set. Inclusive start, exclusive end.
func (s *Set[V]) SetRange(start, end V) {
	for k := start; k < end; k++ {
		s.uint64 |= 1 << k
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

	ss := make([]string, 0, bits.UintSize)
	for v := V(0); v < bits.UintSize; v++ {
		if s.Contains(v) {
			ss = append(ss, say(v))
		}
	}

	return fmt.Sprintf("[%v]", strings.Join(ss, " "))
}
