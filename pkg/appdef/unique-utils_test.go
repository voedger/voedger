/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_duplicates(t *testing.T) {
	require := require.New(t)

	require.Negative(duplicates([]string{"a"}))
	require.Negative(duplicates([]string{"a", "b"}))
	require.Negative(duplicates([]int{0, 1, 2}))

	i, j := duplicates([]int{0, 1, 0})
	require.True(0 == i && 2 == j)

	i, j = duplicates([]int{0, 1, 2, 1})
	require.True(1 == i && 3 == j)

	i, j = duplicates([]bool{true, true})
	require.True(0 == i && 1 == j)

	i, j = duplicates([]string{"a", "b", "c", "c"})
	require.True(2 == i && 3 == j)
}

func Test_subSet(t *testing.T) {
	require := require.New(t)

	t.Run("check empty slices", func(t *testing.T) {
		require.True(subSet([]int{}, []int{}))
		require.True(subSet(nil, []string{}))
		require.True(subSet([]bool{}, nil))
		require.True(subSet[int](nil, nil))

		require.True(subSet(nil, []string{"a", "b"}))
		require.True(subSet([]bool{}, []bool{true, false}))
	})

	t.Run("must be true", func(t *testing.T) {
		require.True(subSet([]int{1}, []int{1}))
		require.True(subSet([]string{"a"}, []string{"a", "b"}))
		require.True(subSet([]int{1, 2, 3}, []int{0, 1, 2, 3, 4}))
	})

	t.Run("must be false", func(t *testing.T) {
		require.False(subSet([]int{1}, []int{}))
		require.False(subSet([]string{"a"}, []string{"b", "c"}))
		require.False(subSet([]int{1, 2, 3}, []int{0, 2, 4, 6, 8}))
	})
}

func Test_overlaps(t *testing.T) {
	require := require.New(t)

	t.Run("check empty slices", func(t *testing.T) {
		require.True(overlaps([]int{}, []int{}))
		require.True(overlaps(nil, []string{}))
		require.True(overlaps([]bool{}, nil))
		require.True(overlaps[int](nil, nil))

		require.True(overlaps(nil, []string{"a", "b"}))
		require.True(overlaps([]bool{true, false}, []bool{}))
	})

	t.Run("must be true", func(t *testing.T) {
		require.True(overlaps([]int{1}, []int{1}))
		require.True(overlaps([]string{"a"}, []string{"a", "b"}))
		require.True(overlaps([]int{0, 1, 2, 3, 4}, []int{1, 2, 3}))
	})

	t.Run("must be false", func(t *testing.T) {
		require.False(overlaps([]int{1}, []int{2}))
		require.False(overlaps([]string{"a"}, []string{"b", "c"}))
		require.False(overlaps([]int{1, 2, 3}, []int{7, 0, 3, 2, 0, -1}))
	})
}

func Test_generateUniqueName(t *testing.T) {
	def := newDef(nil, NewQName("test", "user"), DefKind_CRecord)

	tests := []struct {
		name   string
		fields []string
		want   string
	}{
		{"single field test", []string{"eMail"}, "userUniqueEMail"},
		{"multiply fields test", []string{"field1", "field2"}, "userUnique01"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := generateUniqueName(def, tt.fields); got != tt.want {
				t.Errorf("generateUniqueName(%v, %#v) = %v, want %v", def.QName(), tt.fields, got, tt.want)
			}
		})
	}

	t.Run("too many uniques (> 100) test", func(t *testing.T) {
		require := require.New(t)

		def := newDef(nil, NewQName("test", "rec"), DefKind_CRecord)
		for i := 1; i < 100; i++ {
			def.AddField("i"+strconv.Itoa(i), DataKind_int32, false)
			def.AddField("b"+strconv.Itoa(i), DataKind_bool, false)
		}
		for i := 1; i < 100; i++ {
			def.AddUnique("", []string{"i" + strconv.Itoa(i), "b" + strconv.Itoa(i)})
		}

		require.Panics(func() {
			def.AddUnique("", []string{"i01", "b99"})
		})
	})
}
