/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package set_test

import (
	"fmt"

	"github.com/voedger/voedger/pkg/goutils/set"
)

type Month uint8

const (
	Jan Month = iota
	Feb
	Mar
	Apr
	May
	Jun
	Jul
	Aug
	Sep
	Oct
	Nov
	Dec
)

var MonthStr = map[Month]string{
	Jan: "Jan",
	Feb: "Feb",
	Mar: "Mar",
	Apr: "Apr",
	May: "May",
	Jun: "Jun",
	Jul: "Jul",
	Aug: "Aug",
	Sep: "Sep",
	Oct: "Oct",
	Nov: "Nov",
	Dec: "Dec",
}

func (t Month) String() string {
	if s, ok := MonthStr[t]; ok {
		return s
	}
	return fmt.Sprintf("Month(%d)", t)
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
	s := set.From(Jan, Feb, Mar)
	fmt.Println(s.AsArray())

	// Output:
	// [Jan Feb Mar]
}

func ExampleSet_All() {
	// This example demonstrates how to use Set type.

	// Create new Set from values.
	s := set.From(Jan, Feb, Mar)

	// Enumerate values from Set.
	for i, v := range s.All() {
		fmt.Println(i, v)
	}

	// Output:
	// 0 Jan
	// 1 Feb
	// 2 Mar
}

func ExampleSet_AsArray() {
	// This example demonstrates how to use Set type.

	// Create new Set from values.
	s := set.From(Jan, Feb, Mar)

	// Receive values from Set as array.
	fmt.Println(s.AsArray())

	// Output:
	// [Jan Feb Mar]
}

func ExampleSet_AsBytes() {
	// This example demonstrates how to use Set type.

	// Create new Set from values.
	s := set.From(Jan, Feb, Mar)

	// Receive Set as big-endian bytes.
	fmt.Printf("%b", s.AsBytes())

	// Output:
	// [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 111]
}

func ExampleSet_Chunk() {
	// This example demonstrates how to use Set type.

	// Create new Set from values.
	var year = func() set.Set[Month] {
		y := set.From(Jan, Feb, Mar, Apr, May, Jun, Jul, Aug, Sep, Oct, Nov, Dec)
		y.SetReadOnly()
		return y
	}()

	// Enumerate year by quarter.
	for q := range year.Chunk(3) {
		fmt.Println(q)
	}

	// Output:
	// [Jan Feb Mar]
	// [Apr May Jun]
	// [Jul Aug Sep]
	// [Oct Nov Dec]
}

func ExampleSet_Clear() {
	// This example demonstrates how to use Set type.

	// Create new Set from values.
	s := set.From(Jan, Feb, Mar)

	// Clear specified values from Set.
	s.Clear(Jan)
	fmt.Println(s)

	// Output:
	// [Feb Mar]
}

func ExampleSet_ClearAll() {
	// This example demonstrates how to use Set type.

	// Create new Set from values.
	s := set.From(Jan, Feb, Mar)

	// Clear all values from Set.
	s.ClearAll()
	fmt.Println(s)

	// Output:
	// []
}

func ExampleSet_Collect() {
	// This example demonstrates how to use Set type.

	// Create new Set from values.
	spring := set.From(Mar, Apr, May)
	autumn := set.From(Sep, Oct, Nov)

	// Collect values from iterator into Set.
	offSeason := set.Empty[Month]()
	offSeason.Collect(spring.Values())
	offSeason.Collect(autumn.Values())
	fmt.Println(offSeason)

	// Output:
	// [Mar Apr May Sep Oct Nov]
}

func ExampleSet_Contains() {
	// This example demonstrates how to use Set type.

	// Create new Set from values.
	s := set.From(Jan, Feb, Mar)

	// Check if Set contains specified value.
	fmt.Println(s.Contains(Jan))
	fmt.Println(s.Contains(Nov))

	// Output:
	// true
	// false
}

func ExampleSet_ContainsAll() {
	// This example demonstrates how to use Set type.

	// Create new Set from values.
	s := set.From(Jan, Feb, Mar)

	// Check if Set contains all specified values.
	fmt.Println(s.ContainsAll(Jan, Mar))
	fmt.Println(s.ContainsAll(Jan, Nov))

	// Output:
	// true
	// false
}

func ExampleSet_ContainsAny() {
	// This example demonstrates how to use Set type.
	//
	// Create new Set from values.
	s := set.From(Jan, Feb, Mar)

	// Check if Set contains at least one of specified values.
	fmt.Println(s.ContainsAny(Jan, Mar))
	fmt.Println(s.ContainsAny(Nov, Dec))

	// Output:
	// true
	// false
}

func ExampleSet_First() {
	// This example demonstrates how to use Set type.

	// Create new Set from values.
	s := set.From(Jan, Feb, Mar)

	// Get first value from Set.
	fmt.Println(s.First())

	// Output:
	// Jan true
}

func ExampleSet_Len() {
	// This example demonstrates how to use Set type.

	// Create new Set from values.
	s := set.From(Jan, Feb, Mar)

	// Get count of values in Set.
	fmt.Println(s.Len())

	// Output:
	// 3
}

func ExampleSet_SetRange() {
	// This example demonstrates how to use Set type.

	// Create new Set from values.
	s := set.From(Jan, Feb, Mar)

	// Set range of values to Set.
	s.SetRange(Jul, Oct)
	fmt.Println(s)

	// Output:
	// [Jan Feb Mar Jul Aug Sep]
}

func ExampleSet_Values() {
	// This example demonstrates how to use Set type.

	// Create new Set from values.
	s := set.From(Jan, Feb, Mar)

	// Enumerate values from Set.
	for v := range s.Values() {
		fmt.Println(v)
	}

	// Output:
	// Jan
	// Feb
	// Mar
}
