/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 *
 * @author Daniil Solovyov
 */

package query2

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_getCombinations(t *testing.T) {
	t.Run("1x1", func(t *testing.T) {
		combinations := getCombinations([][]interface{}{{1}})

		require.Equal(t, [][]interface{}{{1}}, combinations)
	})
	t.Run("1x2", func(t *testing.T) {
		combinations := getCombinations([][]interface{}{{1, 2}})

		require.Equal(t, [][]interface{}{{1}, {2}}, combinations)
	})
	t.Run("2x2", func(t *testing.T) {
		combinations := getCombinations([][]interface{}{
			{1, 2},
			{"a", "b"},
		})

		require.Equal(t, [][]interface{}{
			{1, "a"},
			{1, "b"},
			{2, "a"},
			{2, "b"},
		}, combinations)
	})
	t.Run("1x2x3", func(t *testing.T) {
		combinations := getCombinations([][]interface{}{
			{"a"},
			{true, false},
			{1, 2, 3},
		})

		require.Equal(t, [][]interface{}{
			{"a", true, 1},
			{"a", true, 2},
			{"a", true, 3},
			{"a", false, 1},
			{"a", false, 2},
			{"a", false, 3},
		}, combinations)
	})
	t.Run("1x3x2", func(t *testing.T) {
		combinations := getCombinations([][]interface{}{
			{"a"},
			{1, 2, 3},
			{true, false},
		})

		require.Equal(t, [][]interface{}{
			{"a", 1, true},
			{"a", 1, false},
			{"a", 2, true},
			{"a", 2, false},
			{"a", 3, true},
			{"a", 3, false},
		}, combinations)
	})
	t.Run("2x1x3", func(t *testing.T) {
		combinations := getCombinations([][]interface{}{
			{true, false},
			{"a"},
			{1, 2, 3},
		})

		require.Equal(t, [][]interface{}{
			{true, "a", 1},
			{true, "a", 2},
			{true, "a", 3},
			{false, "a", 1},
			{false, "a", 2},
			{false, "a", 3},
		}, combinations)
	})
	t.Run("2х3х1", func(t *testing.T) {
		combinations := getCombinations([][]interface{}{
			{true, false},
			{1, 2, 3},
			{"a"},
		})

		require.Equal(t, [][]interface{}{
			{true, 1, "a"},
			{true, 2, "a"},
			{true, 3, "a"},
			{false, 1, "a"},
			{false, 2, "a"},
			{false, 3, "a"},
		}, combinations)
	})
	t.Run("3x1x2", func(t *testing.T) {
		combinations := getCombinations([][]interface{}{
			{1, 2, 3},
			{"a"},
			{true, false},
		})

		require.Equal(t, [][]interface{}{
			{1, "a", true},
			{1, "a", false},
			{2, "a", true},
			{2, "a", false},
			{3, "a", true},
			{3, "a", false},
		}, combinations)
	})
	t.Run("3x2x1", func(t *testing.T) {
		combinations := getCombinations([][]interface{}{
			{1, 2, 3},
			{true, false},
			{"a"},
		})

		require.Equal(t, [][]interface{}{
			{1, true, "a"},
			{1, false, "a"},
			{2, true, "a"},
			{2, false, "a"},
			{3, true, "a"},
			{3, false, "a"},
		}, combinations)
	})
	t.Run("3x2x4", func(t *testing.T) {
		combinations := getCombinations([][]interface{}{
			{2021, 2022, 2023},
			{time.January, time.April},
			{10, 12, 17, 23},
		})

		require.Equal(t, [][]interface{}{
			{2021, time.January, 10},
			{2021, time.January, 12},
			{2021, time.January, 17},
			{2021, time.January, 23},
			{2021, time.April, 10},
			{2021, time.April, 12},
			{2021, time.April, 17},
			{2021, time.April, 23},
			{2022, time.January, 10},
			{2022, time.January, 12},
			{2022, time.January, 17},
			{2022, time.January, 23},
			{2022, time.April, 10},
			{2022, time.April, 12},
			{2022, time.April, 17},
			{2022, time.April, 23},
			{2023, time.January, 10},
			{2023, time.January, 12},
			{2023, time.January, 17},
			{2023, time.January, 23},
			{2023, time.April, 10},
			{2023, time.April, 12},
			{2023, time.April, 17},
			{2023, time.April, 23},
		}, combinations)
	})
}
func Test_splitPath(t *testing.T) {
	tests := []struct {
		path string
		want []string
	}{
		{path: `name`, want: []string{"name"}},
		{path: `obj.name`, want: []string{"obj", "name"}},
		{path: `"foo.bar".baz`, want: []string{"foo.bar", "baz"}},
		{path: `part1."inner.part".part2`, want: []string{"part1", "inner.part", "part2"}},
		{path: `a."b.c.d".e.f`, want: []string{"a", "b.c.d", "e", "f"}},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s to %s", tt.path, tt.want), func(t *testing.T) {
			require.Equal(t, tt.want, splitPath(tt.path))
		})
	}
}
