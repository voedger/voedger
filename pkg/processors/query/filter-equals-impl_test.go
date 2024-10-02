/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package queryprocessor

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

func TestEqualsFilter_IsMatch(t *testing.T) {
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
			return &EqualsFilter{
				field: "age",
				value: json.Number(strconv.Itoa(age)),
			}
		}
		t.Run("Should match", func(t *testing.T) {
			require.True(t, match(ageFilter(42).IsMatch(fk, row(42))))
		})
		t.Run("Should not match", func(t *testing.T) {
			require.False(t, match(ageFilter(45).IsMatch(fk, row(42))))
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
			return &EqualsFilter{
				field: "age",
				value: json.Number(strconv.Itoa(age)),
			}
		}
		t.Run("Should match", func(t *testing.T) {
			require.True(t, match(ageFilter(42).IsMatch(fk, row(42))))
		})
		t.Run("Should not match", func(t *testing.T) {
			require.False(t, match(ageFilter(45).IsMatch(fk, row(42))))
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
			return &EqualsFilter{
				field:   "height",
				value:   json.Number(fmt.Sprint(height)),
				epsilon: 0.0000001,
			}
		}
		t.Run("Should match", func(t *testing.T) {
			require.True(t, match(heightFilter(42.7).IsMatch(fk, row(42.7))))
		})
		t.Run("Should not match", func(t *testing.T) {
			require.False(t, match(heightFilter(42.71).IsMatch(fk, row(42.7))))
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
			return &EqualsFilter{
				field:   "height",
				value:   json.Number(fmt.Sprint(height)),
				epsilon: 0.0000001,
			}
		}
		t.Run("Should match", func(t *testing.T) {
			require.True(t, match(heightFilter(42.7).IsMatch(fk, row(42.7))))
		})
		t.Run("Should not match", func(t *testing.T) {
			require.False(t, match(heightFilter(42.71).IsMatch(fk, row(42.7))))
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
			return &EqualsFilter{
				field: "name",
				value: name,
			}
		}
		t.Run("Should match", func(t *testing.T) {
			require.True(t, match(nameFilter("Cola").IsMatch(fk, row("Cola"))))
		})
		t.Run("Should not match", func(t *testing.T) {
			require.False(t, match(nameFilter("Beer").IsMatch(fk, row("Cola"))))
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
			return &EqualsFilter{
				field: "active",
				value: active,
			}
		}
		t.Run("Should match", func(t *testing.T) {
			require.True(t, match(activeFilter(true).IsMatch(fk, row(true))))
		})
		t.Run("Should not match", func(t *testing.T) {
			require.False(t, match(activeFilter(true).IsMatch(fk, row(false))))
		})
	})
	t.Run("Should return false on null data type", func(t *testing.T) {
		filter := &EqualsFilter{}

		match, err := filter.IsMatch(nil, nil)

		require.NoError(t, err)
		require.False(t, match)
	})
	t.Run("Should return error on wrong data type", func(t *testing.T) {
		filter := &EqualsFilter{field: "image"}

		match, err := filter.IsMatch(map[string]appdef.DataKind{"image": appdef.DataKind_bytes}, nil)

		require.ErrorIs(t, err, ErrWrongType)
		require.False(t, match)
	})
	t.Run("Compare istructs.RecordID", func(t *testing.T) {
		row := func(id istructs.RecordID) IOutputRow {
			r := &testOutputRow{fields: []string{"id"}}
			r.Set("id", id)
			return r
		}
		fk := FieldsKinds{"id": appdef.DataKind_RecordID}
		ageFilter := func(id istructs.RecordID) IFilter {
			return &EqualsFilter{
				field: "id",
				value: json.Number(strconv.Itoa(int(id))),
			}
		}
		t.Run("Should match", func(t *testing.T) {
			require.True(t, match(ageFilter(42).IsMatch(fk, row(42))))
		})
		t.Run("Should not match", func(t *testing.T) {
			require.False(t, match(ageFilter(45).IsMatch(fk, row(42))))
		})
	})
	t.Run("Compare appdef.QName", func(t *testing.T) {
		row := func(name string) IOutputRow {
			r := &testOutputRow{fields: []string{"sys.QName"}}
			r.Set("sys.QName", name)
			return r
		}
		fk := FieldsKinds{"sys.QName": appdef.DataKind_QName}
		nameFilter := func(name string) IFilter {
			return &EqualsFilter{
				field: "sys.QName",
				value: name,
			}
		}
		tests := []struct {
			name     string
			qNameOne string
			qNameTwo string
			want     bool
		}{
			{
				name:     "Should match",
				qNameOne: "pkg.Entity",
				qNameTwo: "pkg.Entity",
				want:     true,
			},
			{
				name:     "Should not match",
				qNameOne: "pkg.Entity",
				qNameTwo: "pkg1.Entity",
				want:     false,
			},
			{
				name:     "Should not match",
				qNameOne: "pkg.Entity",
				qNameTwo: "pkg.Entity1",
				want:     false,
			},
			{
				name:     "Should not match",
				qNameOne: "pkg.Entity",
				qNameTwo: "pkg1.Entity1",
				want:     false,
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				require.Equal(t, test.want, match(nameFilter(test.qNameOne).IsMatch(fk, row(test.qNameTwo))))
			})
		}
	})
}
