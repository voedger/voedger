/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package mock

import (
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
)

func TestDef(t *testing.T) {
	qNames := struct{ element, obj appdef.QName }{
		appdef.NewQName("test", "el"),
		appdef.NewQName("test", "obj"),
	}

	el := NewDef(qNames.element, appdef.DefKind_Element,
		NewField("f1", appdef.DataKind_string, true),
	)
	obj := NewDef(qNames.obj, appdef.DefKind_Object,
		NewField("f2", appdef.DataKind_int64, true),
		NewVerifiedField("f3", appdef.DataKind_string, false, appdef.VerificationKind_Phone),
	)
	obj.AddContainer(NewContainer("c1", el.QName(), 0, appdef.Occurs_Unbounded))

	appDef := NewAppDef(
		el,
		obj,
	)

	t.Run("test result", func(t *testing.T) {
		require := require.New(t)

		require.Equal(2, appDef.DefCount())
		require.Equal(appDef.DefCount(), func() int {
			cnt := 0
			appDef.Defs(func(appdef.IDef) { cnt++ })
			return cnt
		}())

		t.Run("test element", func(t *testing.T) {
			el := appDef.Def(qNames.element)
			require.NotNil(el)
			require.Equal(qNames.element, el.QName())
			require.Equal(appdef.DefKind_Element, el.Kind())
			require.Equal(appDef, el.App())
			require.Equal(1, el.FieldCount())
			require.NotNil(el.Field("f1"))

			require.Nil(el.Field("unknownField"))
		})

		t.Run("test object", func(t *testing.T) {
			obj := appDef.DefByName(qNames.obj)
			require.NotNil(obj)
			require.Equal(2, obj.FieldCount())

			f2 := obj.Field("f2")
			require.NotNil(f2)
			require.Equal("f2", f2.Name())
			require.Equal(appdef.DataKind_int64, f2.DataKind())
			require.True(f2.Required())
			require.False(f2.IsSys())
			require.True(f2.IsFixedWidth())

			f3 := obj.Field("f3")
			require.NotNil(f3)
			require.True(f3.Verifiable())
			require.True(f3.VerificationKind(appdef.VerificationKind_Phone))
			require.False(f3.VerificationKind(appdef.VerificationKind_EMail))

			require.Equal(1, obj.ContainerCount())
			c1 := obj.Container("c1")
			require.NotNil(c1)
			require.Equal("c1", c1.Name())
			require.Equal(qNames.element, c1.Def())
			require.EqualValues(0, c1.MinOccurs())
			require.Equal(appdef.Occurs_Unbounded, c1.MaxOccurs())
			require.False(c1.IsSys())

			el := obj.ContainerDef("c1")
			require.NotNil(el)
			require.Equal(qNames.element, el.QName())
			require.Equal(appdef.DefKind_Element, el.Kind())

			require.Equal(obj.ContainerCount(), func() int {
				cnt := 0
				obj.Containers(func(appdef.IContainer) { cnt++ })
				return cnt
			}())

			require.Nil(obj.Container("unknownContainer"))
			d := obj.ContainerDef("unknownContainer")
			require.NotNil(d)
			require.Equal(appdef.DefKind_null, d.Kind())
		})

		require.Equal(appdef.DefKind_null, appDef.Def(appdef.NewQName("test", "unknown")).Kind())
		require.Nil(appDef.DefByName(appdef.NewQName("test", "unknown")))
	})
}

func TestUniques(t *testing.T) {
	qName := appdef.NewQName("test", "el")

	def := NewDef(qName, appdef.DefKind_Element,
		NewField("f1", appdef.DataKind_string, true),
		NewField("f2", appdef.DataKind_string, true),
	)
	def.AddUnique(NewUnique("elUnique01", []string{"f1", "f2"}))

	appDef := NewAppDef(def)

	t.Run("test result", func(t *testing.T) {
		require := require.New(t)

		d := appDef.Def(qName)
		require.Equal(1, d.UniqueCount())

		u := d.UniqueByName("elUnique01")
		require.Len(u.Fields(), 2)
		require.Equal("f1", u.Fields()[0].Name())
		require.Equal("f2", u.Fields()[1].Name())

		require.Equal(d.UniqueCount(), func() int {
			cnt := 0
			d.Uniques(func(u appdef.IUnique) {
				cnt++
				switch u.Name() {
				case "elUnique01":
					require.Len(u.Fields(), 2)
					require.Equal("f1", u.Fields()[0].Name())
					require.Equal("f2", u.Fields()[1].Name())
				default:
					require.Failf("unexpected unique %v", u.Name())
				}
			})
			return cnt
		}())

		require.Nil(d.UniqueByName("unknownUnique"))

		t.Run("test id", func(t *testing.T) {
			testID := appdef.UniqueID(100500)
			u := d.UniqueByName("elUnique01")
			u.SetID(testID)

			require.Equal(testID, u.ID())
		})
	})
}

func TestInheritsMockingDef(t *testing.T) {
	fld := Field{}
	fld.
		On("Name").Return("mockField").
		On("VerificationKind", mock.AnythingOfType("appdef.VerificationKind")).Return(true)

	cont := Container{}
	cont.
		On("Name").Return("mockContainer")

	unique := Unique{}
	unique.
		On("Name").Return("mockUnique").
		On("Fields").Return([]appdef.IField{&fld})

	app := AppDef{}

	def := Def{}
	def.
		On("App").Return(&app).
		On("FieldCount").Return(1).
		On("Field", mock.AnythingOfType("string")).Return(&fld).
		On("Fields", mock.AnythingOfType("func(appdef.IField)")).
		Run(func(args mock.Arguments) {
			cb := args.Get(0).(func(appdef.IField))
			cb(&fld)
		}).
		On("ContainerCount").Return(1).
		On("Container", mock.AnythingOfType("string")).Return(&cont).
		On("Containers", mock.AnythingOfType("func(appdef.IContainer)")).
		Run(func(args mock.Arguments) {
			cb := args.Get(0).(func(appdef.IContainer))
			cb(&cont)
		}).
		On("ContainerDef", mock.AnythingOfType("string")).Return(&def).
		On("Kind").Return(appdef.DefKind_CRecord).
		On("QName").Return(appdef.NewQName("test", "cRec")).
		On("Singleton").Return(true).
		On("UniqueByName", mock.AnythingOfType("string")).Return(&unique).
		On("UniqueCount").Return(1).
		On("Uniques", mock.AnythingOfType("func(appdef.IUnique)")).
		Run(func(args mock.Arguments) {
			cb := args.Get(0).(func(appdef.IUnique))
			cb(&unique)
		})

	require := require.New(t)

	require.Equal(&app, def.App())

	require.Equal(1, def.FieldCount())
	f := def.Field("mockField")
	require.NotNil(f)
	require.Equal("mockField", f.Name())
	require.True(f.VerificationKind(appdef.VerificationKind_EMail))
	require.Equal(def.FieldCount(), func() int {
		cnt := 0
		def.Fields(func(appdef.IField) { cnt++ })
		return cnt
	}())

	require.Equal(1, def.ContainerCount())
	c := def.Container("mockContainer")
	require.NotNil(c)
	require.Equal("mockContainer", c.Name())
	require.Equal(def.ContainerCount(), func() int {
		cnt := 0
		def.Containers(func(appdef.IContainer) { cnt++ })
		return cnt
	}())
	require.Equal(&def, def.ContainerDef("mockContainer"))

	require.True(def.Singleton())

	require.Equal(1, def.UniqueCount())
	u := def.UniqueByName("cRecUniqueMockField")
	require.Len(u.Fields(), 1)
	require.Equal("mockField", u.Fields()[0].Name())
	require.Equal(def.UniqueCount(), func() int {
		cnt := 0
		def.Uniques(func(appdef.IUnique) { cnt++ })
		return cnt
	}())
}
