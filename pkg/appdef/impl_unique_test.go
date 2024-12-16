/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"
	"testing"

	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_def_AddUnique(t *testing.T) {
	require := require.New(t)

	qName := NewQName("test", "user")
	un1 := UniqueQName(qName, "EMail")
	un2 := UniqueQName(qName, "Full")

	adb := New()
	adb.AddPackage("test", "test.com/test")

	ws := adb.AddWorkspace(NewQName("test", "workspace"))

	doc := ws.AddCDoc(qName)
	require.NotNil(doc)

	doc.
		AddField("name", DataKind_string, true).
		AddField("surname", DataKind_string, false).
		AddField("lastName", DataKind_string, false).
		AddField("birthday", DataKind_int64, false).
		AddField("sex", DataKind_bool, false).
		AddField("eMail", DataKind_string, false)
	doc.
		AddUnique(un1, []FieldName{"eMail"}).
		AddUnique(un2, []FieldName{"name", "surname", "lastName"})

	t.Run("test is ok", func(t *testing.T) {
		app, err := adb.Build()
		require.NoError(err)

		doc := CDoc(app.Type, qName)
		require.NotEqual(TypeKind_null, doc.Kind())

		require.Equal(2, doc.UniqueCount())

		u := doc.UniqueByName(un2)
		require.Len(u.Fields(), 3)
		require.Equal("lastName", u.Fields()[0].Name())
		require.Equal("name", u.Fields()[1].Name())
		require.Equal("surname", u.Fields()[2].Name())

		require.Equal(doc.UniqueCount(), func() int {
			cnt := 0
			for n, u := range doc.Uniques() {
				cnt++
				require.Equal(n, u.Name())
				switch n {
				case un1:
					require.Len(u.Fields(), 1)
					require.Equal("eMail", u.Fields()[0].Name())
					require.Equal(DataKind_string, u.Fields()[0].DataKind())
				case un2:
					require.Len(u.Fields(), 3)
					require.Equal("lastName", u.Fields()[0].Name())
					require.Equal("name", u.Fields()[1].Name())
					require.Equal("surname", u.Fields()[2].Name())
				}
			}
			return cnt
		}())
	})

	t.Run("test panics", func(t *testing.T) {

		require.Panics(func() {
			doc.AddUnique(NullQName, []FieldName{"sex"})
		}, require.Is(ErrMissedError))

		require.Panics(func() {
			doc.AddUnique(NewQName("naked", "🔫"), []FieldName{"sex"})
		}, require.Is(ErrInvalidError), require.Has("naked.🔫"))

		require.Panics(func() {
			doc.AddUnique(un1, []FieldName{"name", "surname", "lastName"})
		}, require.Is(ErrAlreadyExistsError), require.Has(un1))

		require.Panics(func() {
			doc.AddUnique(qName, []FieldName{"name", "surname", "lastName"})
		}, require.Is(ErrAlreadyExistsError), require.Has(qName))

		require.Panics(func() {
			doc.AddUnique(NewQName("test", "user$uniqueEmpty"), []FieldName{})
		}, require.Is(ErrMissedError))

		require.Panics(func() {
			doc.AddUnique(NewQName("test", "user$uniqueFiledDup"), []FieldName{"birthday", "birthday"})
		}, require.Is(ErrAlreadyExistsError), require.Has("birthday"))

		t.Run("panics if too many fields", func(t *testing.T) {
			adb := New()
			adb.AddPackage("test", "test.com/test")
			ws := adb.AddWorkspace(NewQName("test", "workspace"))
			rec := ws.AddCRecord(NewQName("test", "rec"))
			fldNames := []FieldName{}
			for i := 0; i <= MaxTypeUniqueFieldsCount; i++ {
				n := fmt.Sprintf("f_%#x", i)
				rec.AddField(n, DataKind_bool, false)
				fldNames = append(fldNames, n)
			}
			require.Panics(func() { rec.AddUnique(NewQName("test", "user$uniqueTooLong"), fldNames) },
				require.Is(ErrTooManyError))
		})

		require.Panics(func() {
			doc.AddUnique(NewQName("test", "user$uniqueFieldsSetDup"), []FieldName{"name", "surname", "lastName"})
		}, require.Is(ErrAlreadyExistsError), require.Has(un2))

		require.Panics(func() {
			doc.AddUnique(NewQName("test", "user$uniqueFieldsSetOverlaps"), []FieldName{"surname"})
		}, require.Is(ErrAlreadyExistsError), require.Has(un2))

		require.Panics(func() {
			doc.AddUnique(NewQName("test", "user$uniqueFieldsSetOverlapped"), []FieldName{"eMail", "birthday"})
		}, require.Is(ErrAlreadyExistsError), require.Has(un1))

		require.Panics(func() {
			doc.AddUnique(NewQName("test", "user$uniqueFieldsUnknown"), []FieldName{"unknown"})
		}, require.Is(ErrNotFoundError), require.Has("unknown"))

		t.Run("panics if too many uniques", func(t *testing.T) {
			adb := New()
			adb.AddPackage("test", "test.com/test")
			ws := adb.AddWorkspace(NewQName("test", "workspace"))
			rec := ws.AddCRecord(NewQName("test", "rec"))
			for i := 0; i < MaxTypeUniqueCount; i++ {
				n := fmt.Sprintf("f_%#x", i)
				rec.AddField(n, DataKind_int32, false)
				rec.AddUnique(NewQName("test", "rec$uniques$"+n), []FieldName{n})
			}
			rec.AddField("lastStraw", DataKind_int32, false)
			require.Panics(func() { rec.AddUnique(NewQName("test", "rec$uniques$lastStraw"), []FieldName{"lastStraw"}) },
				require.Is(ErrTooManyError))
		})
	})
}

func Test_type_UniqueField(t *testing.T) {
	// This tests old-style uniques. See [issue #173](https://github.com/voedger/voedger/issues/173)
	require := require.New(t)

	qName := NewQName("test", "user")

	adb := New()
	adb.AddPackage("test", "test.com/test")

	ws := adb.AddWorkspace(NewQName("test", "workspace"))

	doc := ws.AddCDoc(qName)
	require.NotNil(doc)

	doc.
		AddField("name", DataKind_string, true).
		AddField("surname", DataKind_string, false).
		AddField("lastName", DataKind_string, false).
		AddField("birthday", DataKind_int64, false).
		AddField("sex", DataKind_bool, false).
		AddField("eMail", DataKind_string, true)
	doc.SetUniqueField("eMail")

	t.Run("should be ok to build app", func(t *testing.T) {
		app, err := adb.Build()
		require.NoError(err)

		d := CDoc(app.Type, qName)
		require.NotEqual(TypeKind_null, d.Kind())

		fld := d.UniqueField()
		require.Equal("eMail", fld.Name())
		require.True(fld.Required())
	})

	t.Run("should be ok to clear unique field", func(t *testing.T) {
		doc.SetUniqueField("")

		app, err := adb.Build()
		require.NoError(err)

		d := CDoc(app.Type, qName)
		require.NotEqual(TypeKind_null, d.Kind())

		require.Nil(d.UniqueField())
	})

	t.Run("test panics", func(t *testing.T) {
		require.Panics(func() {
			doc.SetUniqueField("naked-🔫")
		}, require.Is(ErrInvalidError), require.Has("naked-🔫"))

		require.Panics(func() {
			doc.SetUniqueField("unknownField")
		}, require.Is(ErrNotFoundError), require.Has("unknownField"))
	})
}

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

	t.Run("should be true", func(t *testing.T) {
		require.True(subSet([]int{1}, []int{1}))
		require.True(subSet([]string{"a"}, []string{"a", "b"}))
		require.True(subSet([]int{1, 2, 3}, []int{0, 1, 2, 3, 4}))
	})

	t.Run("should be false", func(t *testing.T) {
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

	t.Run("should be true", func(t *testing.T) {
		require.True(overlaps([]int{1}, []int{1}))
		require.True(overlaps([]string{"a"}, []string{"a", "b"}))
		require.True(overlaps([]int{0, 1, 2, 3, 4}, []int{1, 2, 3}))
	})

	t.Run("should be false", func(t *testing.T) {
		require.False(overlaps([]int{1}, []int{2}))
		require.False(overlaps([]string{"a"}, []string{"b", "c"}))
		require.False(overlaps([]int{1, 2, 3}, []int{7, 0, 3, 2, 0, -1}))
	})
}
