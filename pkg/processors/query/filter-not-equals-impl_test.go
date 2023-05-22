/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package queryprocessor

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
)

func TestNotEqualsFilter_IsMatch(t *testing.T) {
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
			return &NotEqualsFilter{
				field: "age",
				value: float64(age),
			}
		}
		t.Run("Should match", func(t *testing.T) {
			require.True(t, match(ageFilter(45).IsMatch(fk, row(42))))
		})
		t.Run("Should not match", func(t *testing.T) {
			require.False(t, match(ageFilter(42).IsMatch(fk, row(42))))
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
			return &NotEqualsFilter{
				field: "age",
				value: float64(age),
			}
		}
		t.Run("Should match", func(t *testing.T) {
			require.True(t, match(ageFilter(45).IsMatch(fk, row(42))))
		})
		t.Run("Should not match", func(t *testing.T) {
			require.False(t, match(ageFilter(42).IsMatch(fk, row(42))))
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
			return &NotEqualsFilter{
				field:   "height",
				value:   float64(height),
				epsilon: 0.0000001,
			}
		}
		t.Run("Should match", func(t *testing.T) {
			require.True(t, match(heightFilter(42.71).IsMatch(fk, row(42.7))))
		})
		t.Run("Should not match", func(t *testing.T) {
			require.False(t, match(heightFilter(42.7).IsMatch(fk, row(42.7))))
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
			return &NotEqualsFilter{
				field:   "height",
				value:   height,
				epsilon: 0.0000001,
			}
		}
		t.Run("Should match", func(t *testing.T) {
			require.True(t, match(heightFilter(42.71).IsMatch(fk, row(42.7))))
		})
		t.Run("Should not match", func(t *testing.T) {
			require.False(t, match(heightFilter(42.7).IsMatch(fk, row(42.7))))
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
			return &NotEqualsFilter{
				field: "name",
				value: name,
			}
		}
		t.Run("Should match", func(t *testing.T) {
			require.True(t, match(nameFilter("Beer").IsMatch(fk, row("Cola"))))
		})
		t.Run("Should not match", func(t *testing.T) {
			require.False(t, match(nameFilter("Cola").IsMatch(fk, row("Cola"))))
		})
	})
	t.Run("Compare bool", func(t *testing.T) {
		row := func(active bool) IOutputRow {
			r := &testOutputRow{fields: []string{"active"}}
			r.Set("active", active)
			return r
		}
		fk := FieldsKinds{"active": appdef.DataKind_bool}
		activeFilter := func(active bool) IFilter {
			return &NotEqualsFilter{
				field: "active",
				value: active,
			}
		}
		t.Run("Should match", func(t *testing.T) {
			require.True(t, match(activeFilter(true).IsMatch(fk, row(false))))
		})
		t.Run("Should not match", func(t *testing.T) {
			require.False(t, match(activeFilter(true).IsMatch(fk, row(true))))
		})
	})
	t.Run("Should return false on null data type", func(t *testing.T) {
		filter := &NotEqualsFilter{}

		match, err := filter.IsMatch(nil, nil)

		require.NoError(t, err)
		require.False(t, match)
	})
	t.Run("Should return error on wrong data type", func(t *testing.T) {
		filter := &NotEqualsFilter{field: "image"}

		match, err := filter.IsMatch(map[string]appdef.DataKind{"image": appdef.DataKind_bytes}, nil)

		require.ErrorIs(t, err, ErrWrongType)
		require.False(t, match)
	})
}
