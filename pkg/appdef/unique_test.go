/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_def_AddUnique(t *testing.T) {
	require := require.New(t)

	qName := NewQName("test", "user")
	appDef := New()

	typ := appDef.AddCDoc(qName)
	require.NotNil(typ)

	typ.
		AddField("name", DataKind_string, true).
		AddField("surname", DataKind_string, false).
		AddField("lastName", DataKind_string, false).
		AddField("birthday", DataKind_int64, false).
		AddField("sex", DataKind_bool, false).
		AddField("eMail", DataKind_string, false)
	typ.
		AddUnique("", []string{"eMail"}).
		AddUnique("userUniqueFullName", []string{"name", "surname", "lastName"})

	t.Run("test is ok", func(t *testing.T) {
		app, err := appDef.Build()
		require.NoError(err)

		typ := app.CDoc(qName)
		require.NotEqual(TypeKind_null, typ.Kind())

		require.Equal(2, typ.UniqueCount())

		u := typ.UniqueByName("userUniqueFullName")
		require.Equal(typ, u.ParentType())
		require.Len(u.Fields(), 3)
		require.Equal("lastName", u.Fields()[0].Name())
		require.Equal("name", u.Fields()[1].Name())
		require.Equal("surname", u.Fields()[2].Name())

		require.Equal(typ.UniqueCount(), func() int {
			cnt := 0
			typ.Uniques(func(u IUnique) {
				cnt++
				switch u.Name() {
				case "userUniqueEMail":
					require.Len(u.Fields(), 1)
					require.Equal("eMail", u.Fields()[0].Name())
					require.Equal(DataKind_string, u.Fields()[0].DataKind())
				case "userUniqueFullName":
					require.Len(u.Fields(), 3)
					require.Equal("lastName", u.Fields()[0].Name())
					require.Equal("name", u.Fields()[1].Name())
					require.Equal("surname", u.Fields()[2].Name())
				}
			})
			return cnt
		}())
	})

	t.Run("test unique IDs", func(t *testing.T) {
		id := FirstUniqueID
		typ.Uniques(func(u IUnique) { id++; u.(interface{ SetID(UniqueID) }).SetID(id) })

		require.Nil(typ.UniqueByID(FirstUniqueID))
		require.NotNil(typ.UniqueByID(FirstUniqueID + 1))
		require.NotNil(typ.UniqueByID(FirstUniqueID + 2))
		require.Nil(typ.UniqueByID(FirstUniqueID + 3))
	})

	t.Run("test panics", func(t *testing.T) {

		require.Panics(func() {
			typ.AddUnique("naked-🔫", []string{"sex"})
		}, "panics if invalid unique name")

		require.Panics(func() {
			typ.AddUnique("userUniqueFullName", []string{"name", "surname", "lastName"})
		}, "panics unique with name is already exists")

		t.Run("panics if type kind is not supports uniques", func(t *testing.T) {
			typ := New().AddObject(NewQName("test", "obj"))
			typ.AddField("f1", DataKind_bool, false).AddField("f2", DataKind_bool, false)
			require.Panics(func() {
				typ.(IUniquesBuilder).AddUnique("", []string{"f1", "f2"})
			})
		})

		require.Panics(func() {
			typ.AddUnique("emptyUnique", []string{})
		}, "panics if fields set is empty")

		require.Panics(func() {
			typ.AddUnique("", []string{"birthday", "birthday"})
		}, "if fields has duplicates")

		t.Run("panics if too many fields", func(t *testing.T) {
			rec := New().AddCRecord(NewQName("test", "rec"))
			fldNames := []string{}
			for i := 0; i <= MaxTypeUniqueFieldsCount; i++ {
				n := fmt.Sprintf("f_%#x", i)
				rec.AddField(n, DataKind_bool, false)
				fldNames = append(fldNames, n)
			}
			require.Panics(func() { rec.AddUnique("", fldNames) })
		})

		require.Panics(func() {
			typ.AddUnique("", []string{"name", "surname", "lastName"})
		}, "if fields set is already exists")

		require.Panics(func() {
			typ.AddUnique("", []string{"surname"})
		}, "if fields set overlaps exists")

		require.Panics(func() {
			typ.AddUnique("", []string{"eMail", "birthday"})
		}, "if fields set overlapped by exists")

		require.Panics(func() {
			typ.AddUnique("", []string{"unknown"})
		}, "if fields not exists")

		t.Run("panics if too many uniques", func(t *testing.T) {
			rec := New().AddCRecord(NewQName("test", "rec"))
			for i := 0; i < MaxTypeUniqueCount; i++ {
				n := fmt.Sprintf("f_%#x", i)
				rec.AddField(n, DataKind_int32, false)
				rec.AddUnique("", []string{n})
			}
			rec.AddField("lastStraw", DataKind_int32, false)
			require.Panics(func() { rec.AddUnique("", []string{"lastStraw"}) })
		})
	})
}

func Test_type_UniqueField(t *testing.T) {
	// This tests old-style uniques. See [issue #173](https://github.com/voedger/voedger/issues/173)
	require := require.New(t)

	qName := NewQName("test", "user")
	appDef := New()

	doc := appDef.AddCDoc(qName)
	require.NotNil(doc)

	doc.
		AddField("name", DataKind_string, true).
		AddField("surname", DataKind_string, false).
		AddField("lastName", DataKind_string, false).
		AddField("birthday", DataKind_int64, false).
		AddField("sex", DataKind_bool, false).
		AddField("eMail", DataKind_string, true)
	doc.SetUniqueField("eMail")

	t.Run("test is ok", func(t *testing.T) {
		app, err := appDef.Build()
		require.NoError(err)

		d := app.CDoc(qName)
		require.NotEqual(TypeKind_null, d.Kind())

		fld := d.UniqueField()
		require.Equal("eMail", fld.Name())
		require.True(fld.Required())
	})

	t.Run("must be ok to clear unique field", func(t *testing.T) {
		doc.SetUniqueField("")

		app, err := appDef.Build()
		require.NoError(err)

		d := app.CDoc(qName)
		require.NotEqual(TypeKind_null, d.Kind())

		require.Nil(d.UniqueField())
	})

	t.Run("test panics", func(t *testing.T) {
		require.Panics(func() {
			doc.SetUniqueField("naked-🔫")
		}, "panics if invalid unique field name")

		require.Panics(func() {
			doc.SetUniqueField("unknownField")
		}, "panics if unknown unique field name")
	})
}
