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
	require.True(i == 0 && j == 2)

	i, j = duplicates([]int{0, 1, 2, 1})
	require.True(i == 1 && j == 3)

	i, j = duplicates([]bool{true, true})
	require.True(i == 0 && j == 1)

	i, j = duplicates([]string{"a", "b", "c", "c"})
	require.True(i == 2 && j == 3)
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
	app := newAppDef()
	doc := app.AddCDoc(NewQName("test", "user"))

	tests := []struct {
		name   string
		fields []string
		want   string
	}{
		{"single field test", []string{"eMail"}, "UniqueEMail"},
		{"multiply fields test", []string{"field1", "field2"}, "Unique01"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := generateUniqueName(doc, tt.fields); got != tt.want {
				t.Errorf("generateUniqueName(%v, %#v) = %v, want %v", doc.QName(), tt.fields, got, tt.want)
			}
		})
	}

	t.Run("too many uniques (> 100) test", func(t *testing.T) {
		require := require.New(t)

		appDef := New()
		rec := appDef.AddCRecord(NewQName("test", "rec"))
		for i := 1; i < MaxTypeUniqueCount; i++ {
			rec.AddField("i"+strconv.Itoa(i), DataKind_int32, false)
			rec.AddField("b"+strconv.Itoa(i), DataKind_bool, false)
		}
		for i := 1; i < MaxTypeUniqueCount; i++ {
			rec.AddUnique("", []string{"i" + strconv.Itoa(i), "b" + strconv.Itoa(i)})
		}

		require.Panics(func() {
			rec.AddUnique("", []string{"i01", "b99"})
		})
	})
}
