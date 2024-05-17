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

func testValues(b *testing.B) []byte {
	b.StopTimer()

	values := make([]byte, 0, 256)
	for _, i := range rand.Perm(256) {
		values = append(values, byte(i))
	}

	require.Len(b, values, 256)

	b.StartTimer()
	return values
}

func checkResult(b *testing.B, r []byte) {
	b.StopTimer()

	require.Len(b, r, 256)

	for i := 0; i < 256; i++ {
		require.Equal(b, byte(i), r[i])
	}

	b.StartTimer()
}

func bench_Set(b *testing.B) {
	for i := 0; i < b.N; i++ {
		v := testValues(b)

		set := From(v...)
		result := set.AsArray()

		checkResult(b, result)
	}
}

func bench_Map(b *testing.B) {
	for i := 0; i < b.N; i++ {
		v := testValues(b)

		set := make(map[byte]struct{})
		for _, v := range v {
			set[v] = struct{}{}
		}

		result := make([]byte, 0, len(set))
		for k := range set {
			result = append(result, k)
		}

		slices.Sort(result)

		checkResult(b, result)
	}
}

func bench_Slice(b *testing.B) {
	for i := 0; i < b.N; i++ {
		v := testValues(b)

		set := make([]byte, 0, 256)
		for _, v := range v {
			if !slices.Contains(set, v) {
				set = append(set, v)
			}
		}

		slices.Sort(set)

		checkResult(b, set)
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

func checkClearResult(b *testing.B, r []byte) {
	b.StopTimer()

	require.Len(b, r, 128)

	for i := 0; i < 128; i++ {
		require.Equal(b, 2*byte(i), r[i])
	}

	b.StartTimer()
}

func bench_SetWithClear(b *testing.B) {
	for i := 0; i < b.N; i++ {
		v := testValues(b)

		set := From(v...)
		for _, v := range v {
			if v%2 == 1 {
				set.Clear(v)
			}
		}
		result := set.AsArray()

		checkClearResult(b, result)
	}
}

func bench_MapWithClear(b *testing.B) {
	for i := 0; i < b.N; i++ {
		v := testValues(b)

		set := make(map[byte]struct{})
		for _, v := range v {
			set[v] = struct{}{}
		}

		for _, v := range v {
			if v%2 == 1 {
				delete(set, v)
			}
		}

		result := make([]byte, 0, len(set))
		for k := range set {
			result = append(result, k)
		}

		slices.Sort(result)

		checkClearResult(b, result)
	}
}

func bench_SliceWithClear(b *testing.B) {
	for i := 0; i < b.N; i++ {
		v := testValues(b)

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

		checkClearResult(b, set)
	}
}
