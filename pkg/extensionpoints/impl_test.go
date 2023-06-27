/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package extensionpoints

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type t1 struct {
	fld1 int
}

type t2 struct {
	fld string
}

func TestBasicUsage_ExtensionPoints(t *testing.T) {
	require := require.New(t)
	rep := NewRootExtensionPoint()
	epKey := "testExtensionPoint"
	ep := rep.ExtensionPoint(epKey)

	t.Run("fill the Extension Point", func(t *testing.T) {

		// add standalone value 42
		ep.Add(42)
		require.Panics(func() { ep.Add(42) })

		// add standalone value t1{}
		ep.Add(t1{fld1: 11})
		require.Panics(func() { ep.Add(t1{fld1: 11}) })

		// add value `44` by the key `43`
		ep.AddNamed(43, 44)
		require.Panics(func() { ep.AddNamed(43, 45) })

		// add value t1{} by the key "otherName"
		ep.AddNamed("otherName", t1{fld1: 46})
		require.Panics(func() { ep.AddNamed("otherName", 45) })

		// already have non Extension Point under the key "otherName" -> panic
		require.Panics(func() { ep.ExtensionPoint("otherName") })

		// set new Extension Point as an extension under the "internal" key
		ep.ExtensionPoint("internal")

		// set an object for the Extension Point
		obj := t2{fld: "fld"}
		epInternal := ep.ExtensionPoint("internal", obj)
		epInternal.Add(t1{fld1: 12})

		// set an object again -> panic
		require.Panics(func() { ep.ExtensionPoint("internal", obj) })

		// provide more than one object to set -> panic
		require.Panics(func() { ep.ExtensionPoint("internal", obj, obj) })

		// "internal" key is already exists
		require.Panics(func() { ep.Add("internal") })

	})

	t.Run("use the Extension Point", func(t *testing.T) {
		ep = rep.ExtensionPoint(epKey)
		require.Nil(ep.Value())

		val, ok := ep.Find("unknown")
		require.False(ok)
		require.Nil(val)

		val, ok = ep.Find(42)
		require.True(ok)
		require.EqualValues(42, val)

		val, ok = ep.Find(t1{fld1: 11})
		require.True(ok)
		require.EqualValues(t1{fld1: 11}, val)

		val, ok = ep.Find("otherName")
		require.True(ok)
		require.EqualValues(t1{fld1: 46}, val)

		val, ok = ep.Find("internal")
		require.True(ok)
		epInternal := val.(IExtensionPoint)
		require.Equal(t2{fld: "fld"}, epInternal.Value())
		val, ok = epInternal.Find(t1{fld1: 12})
		require.True(ok)
		require.EqualValues(t1{fld1: 12}, val)

		val, ok = epInternal.Find("unknown")
		require.False(ok)
		require.Nil(val)

		num := 0
		ep.Iterate(func(eKey EKey, value interface{}) {
			// order is preserved
			switch num {
			case 0:
				require.EqualValues(42, eKey)
				require.EqualValues(42, value)
			case 1:
				require.EqualValues(t1{fld1: 11}, eKey)
				require.EqualValues(t1{fld1: 11}, value)
			case 2:
				require.EqualValues(43, eKey)
				require.EqualValues(44, value)
			case 3:
				require.EqualValues("otherName", eKey)
				require.EqualValues(t1{fld1: 46}, value)
			case 4:
				require.EqualValues("internal", eKey)
				epInternal := value.(IExtensionPoint)
				numInt := 0
				epInternal.Iterate(func(eKey EKey, value interface{}) {
					require.EqualValues(t1{fld1: 12}, eKey)
					require.EqualValues(t1{fld1: 12}, value)
					numInt++
				})
				require.Equal(1, numInt)
			}
			num++
		})
		require.Equal(5, num)
	})

}
