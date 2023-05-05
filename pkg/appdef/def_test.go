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

func Test_def_AddField(t *testing.T) {
	require := require.New(t)

	def := New().AddStruct(NewQName("test", "object"), DefKind_Object)
	require.NotNil(def)

	t.Run("must be ok to add field", func(t *testing.T) {
		def.AddField("f1", DataKind_int64, true)

		require.Equal(2, def.FieldCount()) // + sys.QName
		f := def.Field("f1")
		require.NotNil(f)
		require.Equal("f1", f.Name())
		require.False(f.IsSys())

		require.Equal(DataKind_int64, f.DataKind())
		require.True(f.IsFixedWidth())
		require.True(f.DataKind().IsFixed())

		require.True(f.Required())
		require.False(f.Verifiable())
	})

	t.Run("must be panic if empty field name", func(t *testing.T) {
		require.Panics(func() { def.AddField("", DataKind_int64, true) })
	})

	t.Run("must be panic if invalid field name", func(t *testing.T) {
		require.Panics(func() { def.AddField("naked_🔫", DataKind_int64, true) })
		t.Run("but ok if system field", func(t *testing.T) {
			require.NotPanics(func() { def.AddField(SystemField_QName, DataKind_QName, true) })
			require.Equal(2, def.FieldCount())
		})
	})

	t.Run("must be panic if field name dupe", func(t *testing.T) {
		require.Panics(func() { def.AddField("f1", DataKind_int64, true) })
		t.Run("but ok if system field", func(t *testing.T) {
			require.NotPanics(func() { def.AddField(SystemField_QName, DataKind_QName, true) })
			require.Equal(2, def.FieldCount())
		})
	})

	t.Run("must be panic if fields are not allowed by definition kind", func(t *testing.T) {
		view := New().AddView(NewQName("test", "view"))
		def := view.Def()
		require.Panics(func() { def.AddField("f1", DataKind_string, true) })
	})

	t.Run("must be panic if field data kind is not allowed by definition kind", func(t *testing.T) {
		view := New().AddView(NewQName("test", "view"))
		require.Panics(func() { view.AddPartField("f1", DataKind_string) })
	})

	t.Run("must be panic if too many fields", func(t *testing.T) {
		d := New().AddStruct(NewQName("test", "obj"), DefKind_Object)
		for i := 0; i < MaxDefFieldCount-1; i++ { // -1 for sys.QName field
			d.AddField(fmt.Sprintf("f_%#x", i), DataKind_bool, false)
		}
		require.Panics(func() { d.AddField("errorField", DataKind_bool, false) })
	})
}

func Test_def_AddVerifiedField(t *testing.T) {
	require := require.New(t)

	def := New().AddStruct(NewQName("test", "object"), DefKind_Object)
	require.NotNil(def)

	t.Run("must be ok to add verified field", func(t *testing.T) {
		def.AddVerifiedField("f1", DataKind_int64, true, VerificationKind_Phone)
		def.AddVerifiedField("f2", DataKind_int64, true, VerificationKind_Any...)

		require.Equal(3, def.FieldCount()) // + sys.QName
		f1 := def.Field("f1")
		require.NotNil(f1)

		require.True(f1.Verifiable())
		require.False(f1.VerificationKind(VerificationKind_EMail))
		require.True(f1.VerificationKind(VerificationKind_Phone))
		require.False(f1.VerificationKind(VerificationKind_FakeLast))

		f2 := def.Field("f2")
		require.NotNil(f2)

		require.True(f2.Verifiable())
		require.True(f2.VerificationKind(VerificationKind_EMail))
		require.True(f2.VerificationKind(VerificationKind_Phone))
		require.False(f2.VerificationKind(VerificationKind_FakeLast))
	})

	t.Run("must be panic if no verification kinds", func(t *testing.T) {
		require.Panics(func() { def.AddVerifiedField("f3", DataKind_int64, true) })
	})
}

func Test_def_AddContainer(t *testing.T) {
	require := require.New(t)

	appDef := New()
	def := appDef.AddStruct(NewQName("test", "object"), DefKind_Object)
	require.NotNil(def)

	elQName := NewQName("test", "element")
	_ = appDef.AddStruct(elQName, DefKind_Element)

	t.Run("must be ok to add container", func(t *testing.T) {
		def.AddContainer("c1", elQName, 1, Occurs_Unbounded)

		require.Equal(1, def.ContainerCount())
		c := def.Container("c1")
		require.NotNil(c)

		require.Equal("c1", c.Name())
		require.False(c.IsSys())

		require.Equal(elQName, c.Def())
		d := def.ContainerDef("c1")
		require.NotNil(d)
		require.Equal(elQName, d.QName())
		require.Equal(DefKind_Element, d.Kind())

		require.EqualValues(1, c.MinOccurs())
		require.Equal(Occurs_Unbounded, c.MaxOccurs())
	})

	t.Run("must be panic if empty container name", func(t *testing.T) {
		require.Panics(func() { def.AddContainer("", elQName, 1, Occurs_Unbounded) })
	})

	t.Run("must be panic if invalid container name", func(t *testing.T) {
		require.Panics(func() { def.AddContainer("naked_🔫", elQName, 1, Occurs_Unbounded) })
		t.Run("but ok if system container", func(t *testing.T) {
			require.NotPanics(func() { def.AddContainer(SystemContainer_ViewValue, elQName, 1, Occurs_Unbounded) })
			require.Equal(2, def.ContainerCount())
		})
	})

	t.Run("must be panic if container name dupe", func(t *testing.T) {
		require.Panics(func() { def.AddContainer("c1", elQName, 1, Occurs_Unbounded) })
	})

	t.Run("must be panic if invalid occurrences", func(t *testing.T) {
		require.Panics(func() { def.AddContainer("c2", elQName, 1, 0) })
		require.Panics(func() { def.AddContainer("c3", elQName, 2, 1) })
	})

	t.Run("must be panic if containers are not allowed by definition kind", func(t *testing.T) {
		view := appDef.AddView(NewQName("test", "view"))
		pk := view.PartKeyDef()
		require.Panics(func() { pk.AddContainer("c1", elQName, 1, Occurs_Unbounded) })
	})

	t.Run("must be panic if container definition is not compatible", func(t *testing.T) {
		require.Panics(func() { def.AddContainer("c2", def.QName(), 1, 1) })

		d := def.ContainerDef("c2")
		require.NotNil(d)
		require.Equal(DefKind_null, d.Kind())
	})

	t.Run("must be panic if too many containers", func(t *testing.T) {
		qn := NewQName("test", "el")
		d := New().AddStruct(qn, DefKind_Element)
		for i := 0; i < MaxDefContainerCount; i++ {
			d.AddContainer(fmt.Sprintf("c_%#x", i), qn, 0, Occurs_Unbounded)
		}
		require.Panics(func() { d.AddContainer("errorContainer", qn, 0, Occurs_Unbounded) })
	})
}

func Test_def_Singleton(t *testing.T) {
	require := require.New(t)

	appDef := New()

	t.Run("must be ok to create singleton definition", func(t *testing.T) {
		def := appDef.AddStruct(NewQName("test", "singleton"), DefKind_CDoc)
		require.NotNil(def)

		def.SetSingleton()
		require.True(def.Singleton())
	})

	t.Run("must be panic if not CDoc definition", func(t *testing.T) {
		def := appDef.AddStruct(NewQName("test", "wdoc"), DefKind_WDoc)
		require.NotNil(def)

		require.Panics(func() { def.SetSingleton() })

		require.False(def.Singleton())
	})
}

func Test_def_AddUnique(t *testing.T) {
	require := require.New(t)

	qName := NewQName("test", "user")
	appDef := New()

	def := appDef.AddStruct(qName, DefKind_CDoc)
	require.NotNil(def)

	def.
		AddField("name", DataKind_string, true).
		AddField("surname", DataKind_string, false).
		AddField("lastName", DataKind_string, false).
		AddField("birthday", DataKind_int64, false).
		AddField("sex", DataKind_bool, false).
		AddField("eMail", DataKind_string, false).
		AddUnique("", []string{"eMail"}).
		AddUnique("userUniqueFullName", []string{"name", "surname", "lastName"})

	t.Run("test is ok", func(t *testing.T) {
		app, err := appDef.Build()
		require.NoError(err)

		d := app.Def(qName)
		require.NotEqual(DefKind_null, d.Kind())

		require.Equal(2, d.UniqueCount())

		u := d.UniqueByName("userUniqueFullName")
		require.Equal(d, u.Def())
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
		def.Uniques(func(u IUnique) { id++; u.SetID(id) })

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
			d := New().AddStruct(NewQName("test", "obj"), DefKind_Object)
			d.AddField("f1", DataKind_bool, false).AddField("f2", DataKind_bool, false)
			require.Panics(func() {
				d.AddUnique("", []string{"f1", "f2"})
			})
		})

		require.Panics(func() {
			def.AddUnique("emptyUnique", []string{})
		}, "panics if fields set is empty")

		require.Panics(func() {
			def.AddUnique("", []string{"birthday", "birthday"})
		}, "if fields has duplicates")

		t.Run("panics if too many fields", func(t *testing.T) {
			d := New().AddStruct(NewQName("test", "rec"), DefKind_CRecord)
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
			d := New().AddStruct(NewQName("test", "rec"), DefKind_CRecord)
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
