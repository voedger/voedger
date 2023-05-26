/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_def_Singleton(t *testing.T) {
	require := require.New(t)

	appDef := New()

	t.Run("must be ok to create singleton definition", func(t *testing.T) {
		def := appDef.AddCDoc(NewQName("test", "singleton"))
		require.NotNil(def)

		def.SetSingleton()
		require.True(def.Singleton())
	})

	t.Run("must be panic if not CDoc definition", func(t *testing.T) {
		def := appDef.AddWDoc(NewQName("test", "wdoc"))
		require.NotNil(def)

		require.Panics(func() { def.(ICDocBuilder).SetSingleton() })

		require.False(def.(ICDocBuilder).Singleton())
	})
}

func Test_def_AddUnique(t *testing.T) {
	require := require.New(t)

	qName := NewQName("test", "user")
	appDef := New()

	def := appDef.AddCDoc(qName)
	require.NotNil(def)

	def.
		AddField("name", DataKind_string, true).
		AddField("surname", DataKind_string, false).
		AddField("lastName", DataKind_string, false).
		AddField("birthday", DataKind_int64, false).
		AddField("sex", DataKind_bool, false).
		AddField("eMail", DataKind_string, false)
	def.
		AddUnique("", []string{"eMail"}).
		AddUnique("userUniqueFullName", []string{"name", "surname", "lastName"})

	t.Run("test is ok", func(t *testing.T) {
		app, err := appDef.Build()
		require.NoError(err)

		d := app.CDoc(qName)
		require.NotEqual(DefKind_null, d.Kind())

		require.Equal(2, d.UniqueCount())

		u := d.UniqueByName("userUniqueFullName")
		require.Equal(d.QName(), u.Def().QName())
		require.Len(u.Fields(), 3)
		require.Equal("lastName", u.Fields()[0].Name())
		require.Equal("name", u.Fields()[1].Name())
		require.Equal("surname", u.Fields()[2].Name())

		require.Equal(d.UniqueCount(), func() int {
			cnt := 0
			d.Uniques(func(u IUnique) {
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
		def.Uniques(func(u IUnique) { id++; u.(interface{ SetID(UniqueID) }).SetID(id) })

		require.Nil(def.UniqueByID(FirstUniqueID))
		require.NotNil(def.UniqueByID(FirstUniqueID + 1))
		require.NotNil(def.UniqueByID(FirstUniqueID + 2))
		require.Nil(def.UniqueByID(FirstUniqueID + 3))
	})

	t.Run("test panics", func(t *testing.T) {

		require.Panics(func() {
			def.AddUnique("naked-🔫", []string{"sex"})
		}, "panics if invalid unique name")

		require.Panics(func() {
			def.AddUnique("userUniqueFullName", []string{"name", "surname", "lastName"})
		}, "panics unique with name is already exists")

		t.Run("panics if definition kind is not supports uniques", func(t *testing.T) {
			d := New().AddObject(NewQName("test", "obj"))
			d.AddField("f1", DataKind_bool, false).AddField("f2", DataKind_bool, false)
			require.Panics(func() {
				d.(IUniquesBuilder).AddUnique("", []string{"f1", "f2"})
			})
		})

		require.Panics(func() {
			def.AddUnique("emptyUnique", []string{})
		}, "panics if fields set is empty")

		require.Panics(func() {
			def.AddUnique("", []string{"birthday", "birthday"})
		}, "if fields has duplicates")

		t.Run("panics if too many fields", func(t *testing.T) {
			d := New().AddCRecord(NewQName("test", "rec"))
			fldNames := []string{}
			for i := 0; i <= MaxDefUniqueFieldsCount; i++ {
				n := fmt.Sprintf("f_%#x", i)
				d.AddField(n, DataKind_bool, false)
				fldNames = append(fldNames, n)
			}
			require.Panics(func() { d.AddUnique("", fldNames) })
		})

		require.Panics(func() {
			def.AddUnique("", []string{"name", "surname", "lastName"})
		}, "if fields set is already exists")

		require.Panics(func() {
			def.AddUnique("", []string{"surname"})
		}, "if fields set overlaps exists")

		require.Panics(func() {
			def.AddUnique("", []string{"eMail", "birthday"})
		}, "if fields set overlapped by exists")

		require.Panics(func() {
			def.AddUnique("", []string{"unknown"})
		}, "if fields not exists")

		t.Run("panics if too many uniques", func(t *testing.T) {
			d := New().AddCRecord(NewQName("test", "rec"))
			for i := 0; i < MaxDefUniqueCount; i++ {
				n := fmt.Sprintf("f_%#x", i)
				d.AddField(n, DataKind_int32, false)
				d.AddUnique("", []string{n})
			}
			d.AddField("lastStraw", DataKind_int32, false)
			require.Panics(func() { d.AddUnique("", []string{"lastStraw"}) })
		})
	})
}
