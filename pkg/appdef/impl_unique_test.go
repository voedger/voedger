/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef_test

import (
	"fmt"
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_def_AddUnique(t *testing.T) {
	require := require.New(t)

	qName := appdef.NewQName("test", "user")
	un1 := appdef.UniqueQName(qName, "EMail")
	un2 := appdef.UniqueQName(qName, "Full")

	adb := appdef.New()
	adb.AddPackage("test", "test.com/test")

	ws := adb.AddWorkspace(appdef.NewQName("test", "workspace"))

	doc := ws.AddCDoc(qName)
	require.NotNil(doc)

	doc.
		AddField("name", appdef.DataKind_string, true).
		AddField("surname", appdef.DataKind_string, false).
		AddField("lastName", appdef.DataKind_string, false).
		AddField("birthday", appdef.DataKind_int64, false).
		AddField("sex", appdef.DataKind_bool, false).
		AddField("eMail", appdef.DataKind_string, false)
	doc.
		AddUnique(un1, []appdef.FieldName{"eMail"}).
		AddUnique(un2, []appdef.FieldName{"name", "surname", "lastName"})

	t.Run("test is ok", func(t *testing.T) {
		app, err := adb.Build()
		require.NoError(err)

		doc := appdef.CDoc(app.Type, qName)
		require.NotEqual(appdef.TypeKind_null, doc.Kind())

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
					require.Equal(appdef.DataKind_string, u.Fields()[0].DataKind())
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
			doc.AddUnique(appdef.NullQName, []appdef.FieldName{"sex"})
		}, require.Is(appdef.ErrMissedError))

		require.Panics(func() {
			doc.AddUnique(appdef.NewQName("naked", "🔫"), []appdef.FieldName{"sex"})
		}, require.Is(appdef.ErrInvalidError), require.Has("naked.🔫"))

		require.Panics(func() {
			doc.AddUnique(un1, []appdef.FieldName{"name", "surname", "lastName"})
		}, require.Is(appdef.ErrAlreadyExistsError), require.Has(un1))

		require.Panics(func() {
			doc.AddUnique(qName, []appdef.FieldName{"name", "surname", "lastName"})
		}, require.Is(appdef.ErrAlreadyExistsError), require.Has(qName))

		require.Panics(func() {
			doc.AddUnique(appdef.NewQName("test", "user$uniqueEmpty"), []appdef.FieldName{})
		}, require.Is(appdef.ErrMissedError))

		require.Panics(func() {
			doc.AddUnique(appdef.NewQName("test", "user$uniqueFiledDup"), []appdef.FieldName{"birthday", "birthday"})
		}, require.Is(appdef.ErrAlreadyExistsError), require.Has("birthday"))

		t.Run("panics if too many fields", func(t *testing.T) {
			adb := appdef.New()
			adb.AddPackage("test", "test.com/test")
			ws := adb.AddWorkspace(appdef.NewQName("test", "workspace"))
			rec := ws.AddCRecord(appdef.NewQName("test", "rec"))
			fldNames := []appdef.FieldName{}
			for i := 0; i <= appdef.MaxTypeUniqueFieldsCount; i++ {
				n := fmt.Sprintf("f_%#x", i)
				rec.AddField(n, appdef.DataKind_bool, false)
				fldNames = append(fldNames, n)
			}
			require.Panics(func() { rec.AddUnique(appdef.NewQName("test", "user$uniqueTooLong"), fldNames) },
				require.Is(appdef.ErrTooManyError))
		})

		require.Panics(func() {
			doc.AddUnique(appdef.NewQName("test", "user$uniqueFieldsSetDup"), []appdef.FieldName{"name", "surname", "lastName"})
		}, require.Is(appdef.ErrAlreadyExistsError), require.Has(un2))

		require.Panics(func() {
			doc.AddUnique(appdef.NewQName("test", "user$uniqueFieldsSetOverlaps"), []appdef.FieldName{"surname"})
		}, require.Is(appdef.ErrAlreadyExistsError), require.Has(un2))

		require.Panics(func() {
			doc.AddUnique(appdef.NewQName("test", "user$uniqueFieldsSetOverlapped"), []appdef.FieldName{"eMail", "birthday"})
		}, require.Is(appdef.ErrAlreadyExistsError), require.Has(un1))

		require.Panics(func() {
			doc.AddUnique(appdef.NewQName("test", "user$uniqueFieldsUnknown"), []appdef.FieldName{"unknown"})
		}, require.Is(appdef.ErrNotFoundError), require.Has("unknown"))

		t.Run("panics if too many uniques", func(t *testing.T) {
			adb := appdef.New()
			adb.AddPackage("test", "test.com/test")
			ws := adb.AddWorkspace(appdef.NewQName("test", "workspace"))
			rec := ws.AddCRecord(appdef.NewQName("test", "rec"))
			for i := 0; i < appdef.MaxTypeUniqueCount; i++ {
				n := fmt.Sprintf("f_%#x", i)
				rec.AddField(n, appdef.DataKind_int32, false)
				rec.AddUnique(appdef.NewQName("test", "rec$uniques$"+n), []appdef.FieldName{n})
			}
			rec.AddField("lastStraw", appdef.DataKind_int32, false)
			require.Panics(func() {
				rec.AddUnique(appdef.NewQName("test", "rec$uniques$lastStraw"), []appdef.FieldName{"lastStraw"})
			},
				require.Is(appdef.ErrTooManyError))
		})
	})
}

func Test_type_UniqueField(t *testing.T) {
	// This tests old-style uniques. See [issue #173](https://github.com/voedger/voedger/issues/173)
	require := require.New(t)

	qName := appdef.NewQName("test", "user")

	adb := appdef.New()
	adb.AddPackage("test", "test.com/test")

	ws := adb.AddWorkspace(appdef.NewQName("test", "workspace"))

	doc := ws.AddCDoc(qName)
	require.NotNil(doc)

	doc.
		AddField("name", appdef.DataKind_string, true).
		AddField("surname", appdef.DataKind_string, false).
		AddField("lastName", appdef.DataKind_string, false).
		AddField("birthday", appdef.DataKind_int64, false).
		AddField("sex", appdef.DataKind_bool, false).
		AddField("eMail", appdef.DataKind_string, true)
	doc.SetUniqueField("eMail")

	t.Run("should be ok to build app", func(t *testing.T) {
		app, err := adb.Build()
		require.NoError(err)

		d := appdef.CDoc(app.Type, qName)
		require.NotEqual(appdef.TypeKind_null, d.Kind())

		fld := d.UniqueField()
		require.Equal("eMail", fld.Name())
		require.True(fld.Required())
	})

	t.Run("should be ok to clear unique field", func(t *testing.T) {
		doc.SetUniqueField("")

		app, err := adb.Build()
		require.NoError(err)

		d := appdef.CDoc(app.Type, qName)
		require.NotEqual(appdef.TypeKind_null, d.Kind())

		require.Nil(d.UniqueField())
	})

	t.Run("test panics", func(t *testing.T) {
		require.Panics(func() {
			doc.SetUniqueField("naked-🔫")
		}, require.Is(appdef.ErrInvalidError), require.Has("naked-🔫"))

		require.Panics(func() {
			doc.SetUniqueField("unknownField")
		}, require.Is(appdef.ErrNotFoundError), require.Has("unknownField"))
	})
}
