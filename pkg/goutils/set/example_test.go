/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package set_test

import (
	"fmt"
	"strings"

	"github.com/voedger/voedger/pkg/goutils/set"
)

type Month uint8

const (
	Month_jan Month = iota
	Month_feb
	Month_mar
	Month_apr
	Month_may
	Month_jun
	Month_jul
	Month_aug
	Month_sep
	Month_oct
	Month_nov
	Month_dec

	Month_count
)

var TypeKindStr = map[Month]string{
	Month_jan: "Month_jan",
	Month_feb: "Month_feb",
	Month_mar: "Month_mar",
	Month_apr: "Month_apr",
	Month_may: "Month_may",
	Month_jun: "Month_jun",
	Month_jul: "Month_jul",
	Month_aug: "Month_aug",
	Month_sep: "Month_sep",
	Month_oct: "Month_oct",
	Month_nov: "Month_nov",
	Month_dec: "Month_dec",
}

func (t Month) String() string {
	if s, ok := TypeKindStr[t]; ok {
		return s
	}
	return fmt.Sprintf("Month(%d)", t)
}

func (t Month) TrimString() string {
	return strings.TrimPrefix(t.String(), "Month_")
}

func ExampleEmpty() {
	// This example demonstrates how to use Set type.

	// Create new empty Set.
	s := set.Empty[Month]()
	fmt.Println(s)

	// Output:
	// []
}

func ExampleFrom() {
	// This example demonstrates how to use Set type.

	// Create new Set from values.
	s := set.From(Month_jan, Month_feb, Month_mar)
	fmt.Println(s.AsArray())

	// Output:
	// [Month_jan Month_feb Month_mar]
}

func ExampleSet_AsArray() {
	// This example demonstrates how to use Set type.

	// Create new Set from values.
	s := set.From(Month_jan, Month_feb, Month_mar)

	// Receive values from Set as array.
	fmt.Println(s.AsArray())

	// Output:
	// [Month_jan Month_feb Month_mar]
}

func ExampleSet_AsBytes() {
	// This example demonstrates how to use Set type.

	// Create new Set from values.
	s := set.From(Month_jan, Month_feb, Month_mar)

	// Receive Set as big-endian bytes.
	fmt.Printf("%b", s.AsBytes())

	// Output:
	// [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 111]
}

func ExampleSet_Clear() {
	// This example demonstrates how to use Set type.

	// Create new Set from values.
	s := set.From(Month_jan, Month_feb, Month_mar)

	// Clear specified values from Set.
	s.Clear(Month_jan)
	fmt.Println(s)

	// Output:
	// [feb mar]
}

func ExampleSet_ClearAll() {
	// This example demonstrates how to use Set type.

	// Create new Set from values.
	s := set.From(Month_jan, Month_feb, Month_mar)

	// Clear all values from Set.
	s.ClearAll()
	fmt.Println(s)

	// Output:
	// []
}

func ExampleSet_Contains() {
	// This example demonstrates how to use Set type.

	// Create new Set from values.
	s := set.From(Month_jan, Month_feb, Month_mar)

	// Check if Set contains specified value.
	fmt.Println(s.Contains(Month_jan))
	fmt.Println(s.Contains(Month_nov))

	// Output:
	// true
	// false
}

func ExampleSet_ContainsAll() {
	// This example demonstrates how to use Set type.

	// Create new Set from values.
	s := set.From(Month_jan, Month_feb, Month_mar)

	// Check if Set contains all specified values.
	fmt.Println(s.ContainsAll(Month_jan, Month_mar))
	fmt.Println(s.ContainsAll(Month_jan, Month_nov))

	// Output:
	// true
	// false
}

func ExampleSet_ContainsAny() {
	// This example demonstrates how to use Set type.
	//
	// Create new Set from values.
	s := set.From(Month_jan, Month_feb, Month_mar)

	// Check if Set contains at least one of specified values.
	fmt.Println(s.ContainsAny(Month_jan, Month_mar))
	fmt.Println(s.ContainsAny(Month_nov, Month_dec))

	// Output:
	// true
	// false
}

func ExampleSet_Enumerate() {
	// This example demonstrates how to use Set type.

	// Create new Set from values.
	s := set.From(Month_jan, Month_feb, Month_mar)

	// Enumerate values from Set.
	s.Enumerate(func(v Month) {
		fmt.Println(v)
	})

	// Output:
	// Month_jan
	// Month_feb
	// Month_mar
}

func ExampleSet_First() {
	// This example demonstrates how to use Set type.

	// Create new Set from values.
	s := set.From(Month_jan, Month_feb, Month_mar)

	// Get first value from Set.
	fmt.Println(s.First())

	// Output:
	// Month_jan true
}

func ExampleSet_Len() {
	// This example demonstrates how to use Set type.

	// Create new Set from values.
	s := set.From(Month_jan, Month_feb, Month_mar)

	// Get count of values in Set.
	fmt.Println(s.Len())

	// Output:
	// 3
}

func ExampleSet_SetRange() {
	// This example demonstrates how to use Set type.

	// Create new Set from values.
	s := set.From(Month_jan, Month_feb, Month_mar)

	// Set range of values to Set.
	s.SetRange(Month_jul, Month_oct)
	fmt.Println(s)

	// Output:
	// [jan feb mar jul aug sep]
}
