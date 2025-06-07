/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package uniques_test

import (
	"fmt"
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_Uniques(t *testing.T) {
	require := require.New(t)

	docName := appdef.NewQName("test", "doc")
	un1 := appdef.UniqueQName(docName, "EMail")
	un2 := appdef.UniqueQName(docName, "Full")

	var app appdef.IAppDef

	t.Run("should be ok to add document", func(t *testing.T) {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")
		wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))

		doc := wsb.AddCDoc(docName)
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

		a, err := adb.Build()
		require.NoError(err)

		app = a
	})

	t.Run("test is ok", func(t *testing.T) {
		doc := appdef.CDoc(app.Type, docName)

		require.Equal(2, doc.UniqueCount())

		u := doc.UniqueByName(un2)
		ff := u.Fields()
		require.Len(ff, 3)
		require.Equal("lastName", ff[0].Name())
		require.Equal("name", ff[1].Name())
		require.Equal("surname", ff[2].Name())

		require.Equal(doc.UniqueCount(), func() int {
			cnt := 0
			for n, u := range doc.Uniques() {
				cnt++
				require.Equal(n, u.Name())
				ff := u.Fields()
				switch n {
				case un1:
					require.Len(ff, 1)
					require.Equal("eMail", ff[0].Name())
					require.Equal(appdef.DataKind_string, ff[0].DataKind())
				case un2:
					require.Len(ff, 3)
					require.Equal("lastName", ff[0].Name())
					require.Equal("name", ff[1].Name())
					require.Equal("surname", ff[2].Name())
				}
			}
			return cnt
		}())
	})
}

func Test_UniquesPanics(t *testing.T) {
	require := require.New(t)

	docName := appdef.NewQName("test", "doc")
	un1 := appdef.UniqueQName(docName, "EMail")
	un2 := appdef.UniqueQName(docName, "Full")

	adb := builder.New()
	adb.AddPackage("test", "test.com/test")
	wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))

	doc := wsb.AddCDoc(docName)
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

	t.Run("should be panics", func(t *testing.T) {

		require.Panics(func() {
			doc.AddUnique(appdef.NullQName, []appdef.FieldName{"sex"})
		}, require.Is(appdef.ErrMissedError),
			"if missed unique name")

		require.Panics(func() {
			doc.AddUnique(appdef.NewQName("naked", "🔫"), []appdef.FieldName{"sex"})
		}, require.Is(appdef.ErrInvalidError), require.Has("naked.🔫"),
			"if invalid unique name")

		require.Panics(func() {
			doc.AddUnique(un1, []appdef.FieldName{"name", "surname", "lastName"})
		}, require.Is(appdef.ErrAlreadyExistsError), require.Has(un1),
			"if unique name already used")

		require.Panics(func() {
			doc.AddUnique(docName, []appdef.FieldName{"name", "surname", "lastName"})
		}, require.Is(appdef.ErrAlreadyExistsError), require.Has(docName),
			"if unique name used by other package entity")

		require.Panics(func() {
			doc.AddUnique(appdef.NewQName("test", "doc$uuu"), []appdef.FieldName{})
		}, require.Is(appdef.ErrMissedError),
			"if fields missed")

		require.Panics(func() {
			doc.AddUnique(appdef.NewQName("test", "doc$uuu"), []appdef.FieldName{"birthday", "birthday"})
		}, require.Is(appdef.ErrAlreadyExistsError), require.Has("birthday"),
			"if fields with duplicates")

		t.Run("if too many fields", func(t *testing.T) {
			adb := builder.New()
			adb.AddPackage("test", "test.com/test")
			wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))
			rec := wsb.AddCRecord(appdef.NewQName("test", "rec"))
			fldNames := []appdef.FieldName{}
			for i := 0; i <= appdef.MaxTypeUniqueFieldsCount; i++ {
				n := fmt.Sprintf("f_%#x", i)
				rec.AddField(n, appdef.DataKind_bool, false)
				fldNames = append(fldNames, n)
			}
			require.Panics(func() { rec.AddUnique(appdef.NewQName("test", "rec$uuu"), fldNames) },
				require.Is(appdef.ErrTooManyError))
		})

		require.Panics(func() {
			doc.AddUnique(appdef.NewQName("test", "doc$uuu"), []appdef.FieldName{"name", "surname", "lastName"})
		}, require.Is(appdef.ErrAlreadyExistsError), require.Has(un2),
			"if unique with specified fields is already exists")

		require.Panics(func() {
			doc.AddUnique(appdef.NewQName("test", "doc$uuu"), []appdef.FieldName{"surname"})
		}, require.Is(appdef.ErrAlreadyExistsError), require.Has(un2),
			"if unique with specified fields is overlapped by existing")

		require.Panics(func() {
			doc.AddUnique(appdef.NewQName("test", "doc$uuu"), []appdef.FieldName{"eMail", "birthday"})
		}, require.Is(appdef.ErrAlreadyExistsError), require.Has(un1),
			"if unique with specified fields is overlaps existing")

		require.Panics(func() {
			doc.AddUnique(appdef.NewQName("test", "doc$uuu"), []appdef.FieldName{"unknown"})
		}, require.Is(appdef.ErrNotFoundError), require.Has("unknown"),
			"if fields with unknown field")

		t.Run("panics if too many uniques", func(t *testing.T) {
			adb := builder.New()
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

func Test_UniqueField(t *testing.T) {
	// This tests old-style uniques. See [issue #173](https://github.com/voedger/voedger/issues/173)
	require := require.New(t)

	qName := appdef.NewQName("test", "user")

	adb := builder.New()
	adb.AddPackage("test", "test.com/test")

	wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))

	doc := wsb.AddCDoc(qName)
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
