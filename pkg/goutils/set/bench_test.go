/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package set

import (
	"math/rand"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

// Benchmark_BasicUsage benchmarks basic usage of the set implemented via set package, map and slice.
//
// # Basic usage scenario:
//  1. Create a set from a slice of 256 bytes.
//  2. Convert the set to an array.
func Benchmark_BasicUsage(b *testing.B) {
	b.Run("Set", func(b *testing.B) {
		bench_Set(b)
	})

	b.Run("Map", func(b *testing.B) {
		bench_Map(b)
	})

	b.Run("Slice", func(b *testing.B) {
		bench_Slice(b)
	})
}

var testValues = func(b *testing.B) []byte {
	values := make([]byte, 0, 256)
	for _, i := range rand.Perm(256) {
		values = append(values, byte(i))
	}

	require.Len(b, values, 256)

	return values
}

func bench_Set(b *testing.B) {
	v := testValues(b)

	for i := 0; i < b.N; i++ {
		set := From(v...)
		_ = set.AsArray()
	}
}

func bench_Map(b *testing.B) {
	v := testValues(b)

	for i := 0; i < b.N; i++ {
		set := make(map[byte]struct{})
		for _, v := range v {
			set[v] = struct{}{}
		}

		result := make([]byte, 0, len(set))
		for k := range set {
			result = append(result, k)
		}

		slices.Sort(result)
	}
}

func bench_Slice(b *testing.B) {
	v := testValues(b)

	for i := 0; i < b.N; i++ {
		set := make([]byte, 0, 256)
		for _, v := range v {
			if !slices.Contains(set, v) {
				set = append(set, v)
			}
		}

		slices.Sort(set)
	}
}

// Benchmark_WithClear benchmarks basic usage of set plus clearing elements from set.
//
// # Basic usage scenario:
//
//  1. Create a set from a slice of 256 bytes.
//  2. Clear odd values from the set.
//  3. Convert the set to an array.
func Benchmark_WithClear(b *testing.B) {
	b.Run("Set", func(b *testing.B) {
		bench_SetWithClear(b)
	})

	b.Run("Map", func(b *testing.B) {
		bench_MapWithClear(b)
	})

	b.Run("Slice", func(b *testing.B) {
		bench_SliceWithClear(b)
	})
}

func bench_SetWithClear(b *testing.B) {
	v := testValues(b)

	for i := 0; i < b.N; i++ {
		set := From(v...)
		for _, v := range v {
			if v%2 == 1 {
				set.Clear(v)
			}
		}
		_ = set.AsArray()
	}
}

func bench_MapWithClear(b *testing.B) {
	v := testValues(b)

	for i := 0; i < b.N; i++ {
		set := make(map[byte]struct{})
		for _, v := range v {
			set[v] = struct{}{}
		}

		for _, v := range v {
			if v%2 == 1 {
				delete(set, v)
			}
		}

		v := make([]byte, 0, len(set))
		for k := range set {
			v = append(v, k)
		}
	}
}

func bench_SliceWithClear(b *testing.B) {
	v := testValues(b)

	for i := 0; i < b.N; i++ {
		set := make([]byte, 0, 256)
		for _, v := range v {
			if !slices.Contains(set, v) {
				set = append(set, v)
			}
		}

		set = slices.DeleteFunc(set, func(v byte) bool {
			return v%2 == 1
		})

		slices.Sort(set)
	}
}
