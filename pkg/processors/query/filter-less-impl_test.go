/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package queryprocessor

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
)

func TestLessFilter_IsMatch(t *testing.T) {
	match := func(match bool, err error) bool {
		require.NoError(t, err)
		return match
	}
	t.Run("Compare int32", func(t *testing.T) {
		row := func(age int32) IOutputRow {
			r := &testOutputRow{fields: []string{"age"}}
			r.Set("age", age)
			return r
		}
		fk := FieldsKinds{"age": appdef.DataKind_int32}
		ageFilter := func(age int) IFilter {
			return &LessFilter{
				field: "age",
				value: float64(age),
			}
		}
		t.Run("Should match", func(t *testing.T) {
			require.True(t, match(ageFilter(43).IsMatch(fk, row(42))))
		})
		t.Run("Should not match", func(t *testing.T) {
			require.False(t, match(ageFilter(41).IsMatch(fk, row(42))))
		})
	})
	t.Run("Compare int64", func(t *testing.T) {
		row := func(age int64) IOutputRow {
			r := &testOutputRow{fields: []string{"age"}}
			r.Set("age", age)
			return r
		}
		fk := FieldsKinds{"age": appdef.DataKind_int64}
		ageFilter := func(age int) IFilter {
			return &LessFilter{
				field: "age",
				value: float64(age),
			}
		}
		t.Run("Should match", func(t *testing.T) {
			require.True(t, match(ageFilter(43).IsMatch(fk, row(42))))
		})
		t.Run("Should not match", func(t *testing.T) {
			require.False(t, match(ageFilter(41).IsMatch(fk, row(42))))
		})
	})
	t.Run("Compare float32", func(t *testing.T) {
		row := func(height float32) IOutputRow {
			r := &testOutputRow{fields: []string{"height"}}
			r.Set("height", height)
			return r
		}
		fk := FieldsKinds{"height": appdef.DataKind_float32}
		heightFilter := func(height float32) IFilter {
			return &LessFilter{
				field: "height",
				value: float64(height),
			}
		}
		t.Run("Should match", func(t *testing.T) {
			require.True(t, match(heightFilter(42.71).IsMatch(fk, row(42.7))))
		})
		t.Run("Should not match", func(t *testing.T) {
			require.False(t, match(heightFilter(42.69).IsMatch(fk, row(42.7))))
		})
	})
	t.Run("Compare float64", func(t *testing.T) {
		row := func(height float64) IOutputRow {
			r := &testOutputRow{fields: []string{"height"}}
			r.Set("height", height)
			return r
		}
		fk := FieldsKinds{"height": appdef.DataKind_float64}
		heightFilter := func(height float64) IFilter {
			return &LessFilter{
				field: "height",
				value: height,
			}
		}
		t.Run("Should match", func(t *testing.T) {
			require.True(t, match(heightFilter(42.71).IsMatch(fk, row(42.7))))
		})
		t.Run("Should not match", func(t *testing.T) {
			require.False(t, match(heightFilter(42.69).IsMatch(fk, row(42.7))))
		})
	})
	t.Run("Compare string", func(t *testing.T) {
		row := func(name string) IOutputRow {
			r := &testOutputRow{fields: []string{"name"}}
			r.Set("name", name)
			return r
		}
		fk := FieldsKinds{"name": appdef.DataKind_string}
		nameFilter := func(name string) IFilter {
			return &LessFilter{
				field: "name",
				value: name,
			}
		}
		t.Run("Should match", func(t *testing.T) {
			require.True(t, match(nameFilter("Xenta").IsMatch(fk, row("Cola"))))
		})
		t.Run("Should not match", func(t *testing.T) {
			require.False(t, match(nameFilter("Amaretto").IsMatch(fk, row("Cola"))))
		})
	})
	t.Run("Should return false on null data type", func(t *testing.T) {
		filter := &LessFilter{}

		match, err := filter.IsMatch(nil, nil)

		require.NoError(t, err)
		require.False(t, match)
	})
	t.Run("Should return error on wrong data type", func(t *testing.T) {
		filter := &LessFilter{field: "image"}

		match, err := filter.IsMatch(map[string]appdef.DataKind{"image": appdef.DataKind_bytes}, nil)

		require.ErrorIs(t, err, ErrWrongType)
		require.False(t, match)
	})
}
