/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAddView(t *testing.T) {
	require := require.New(t)

	app := New()

	objName := NewQName("test", "object")
	_ = app.AddObject(objName)

	viewName := NewQName("test", "view")
	v := app.AddView(viewName)
	require.Equal(viewName, v.QName())
	require.Equal(DefKind_ViewRecord, v.Kind())
	require.Equal(2, v.ContainerCount()) // key + value

	key := v.Key()
	require.Equal(v.Container(SystemContainer_ViewKey).Def(), key)
	require.Equal(ViewKeyDefName(viewName), key.QName())
	require.Equal(DefKind_ViewRecord_Key, key.Kind())
	require.Equal(2, key.ContainerCount()) // pk + cc

	pk := key.PartKey()
	require.Equal(key.Container(SystemContainer_ViewPartitionKey).Def(), pk)
	require.Equal(ViewPartitionKeyDefName(viewName), pk.QName())
	require.Equal(DefKind_ViewRecord_PartitionKey, pk.Kind())

	cc := key.ClustCols()
	require.Equal(key.Container(SystemContainer_ViewClusteringCols).Def(), cc)
	require.Equal(ViewClusteringColumnsDefName(viewName), cc.QName())
	require.Equal(DefKind_ViewRecord_ClusteringColumns, cc.Kind())

	val := v.Value()
	require.NotNil(val)
	require.Equal(ViewValueDefName(viewName), val.QName())
	require.Equal(DefKind_ViewRecord_Value, val.Kind())

	require.Equal(v, app.View(v.QName()))
	require.Nil(app.View(NewQName("test", "unknown")))

	t.Run("must be ok to add partition key fields", func(t *testing.T) {
		v.AddPartField("pkF1", DataKind_int64)
		v.AddPartField("pkF2", DataKind_bool)
		require.Equal(2, pk.FieldCount())

		t.Run("panic if variable length field added to pk", func(t *testing.T) {
			require.Panics(func() {
				v.AddPartField("pkF3", DataKind_string)
			})
		})
	})

	t.Run("must be ok to add clustering columns fields", func(t *testing.T) {
		v.AddClustColumn("ccF1", DataKind_int64)
		v.AddClustColumn("ccF2", DataKind_QName)
		require.Equal(2, cc.FieldCount())

		t.Run("panic if field already exists in pk", func(t *testing.T) {
			require.Panics(func() {
				v.AddClustColumn("pkF1", DataKind_int64)
			})
		})
	})

	t.Run("must be ok to read view full key", func(t *testing.T) {
		require.Equal(4, key.FieldCount())
		cnt := 0
		key.Fields(func(f IField) {
			cnt++
			switch cnt {
			case 1:
				require.Equal("pkF1", f.Name())
				require.True(f.Required())
			case 2:
				require.Equal("pkF2", f.Name())
				require.True(f.Required())
			case 3:
				require.Equal("ccF1", f.Name())
				require.False(f.Required())
			case 4:
				require.Equal("ccF2", f.Name())
				require.False(f.Required())
			default:
				require.Fail("unexpected field «%s»", f.Name())
			}
		})
		require.Equal(key.FieldCount(), cnt)
	})

	t.Run("must be ok to add value fields", func(t *testing.T) {
		v.AddValueField("valF1", DataKind_bool, true)
		v.AddValueField("valF2", DataKind_string, false)
		require.Equal(2+1, val.FieldCount()) // + sys.QName field
	})

	_, err := app.Build()
	require.NoError(err)

	t.Run("must be ok to cast Def() as IView", func(t *testing.T) {
		a, err := app.Build()
		require.NoError(err)

		d := a.Def(viewName)
		require.NotNil(d)
		require.Equal(DefKind_ViewRecord, d.Kind())

		v, ok := d.(IView)
		require.True(ok)
		require.NotNil(v)

		k, ok := a.Def(ViewKeyDefName(viewName)).(IViewKey)
		require.True(ok)
		require.Equal(k, key)
		require.Equal(k, v.Key())
	})

	t.Run("must be nil if unknown view", func(t *testing.T) {
		a, err := app.Build()
		require.NoError(err)

		v := a.View(NewQName("unknown", "view"))
		require.Nil(v)
	})

	t.Run("must be nil if not view", func(t *testing.T) {
		a, err := app.Build()
		require.NoError(err)

		v := a.View(objName)
		require.Nil(v)

		d := a.Def(objName)
		require.NotNil(d)
		v, ok := d.(IView)
		require.False(ok)
		require.Nil(v)
	})

	t.Run("must be ok to add value fields to view after app build", func(t *testing.T) {
		v.AddValueField("valF3", DataKind_Event, false)
		require.Equal(3+1, val.FieldCount())

		_, err := app.Build()
		require.NoError(err)
	})

	t.Run("must be ok to add pk or cc fields to view after app build", func(t *testing.T) {
		v.AddPartField("pkF3", DataKind_QName)
		v.AddClustColumn("ccF3", DataKind_string)

		require.Equal(3, pk.FieldCount())
		require.Equal(3, cc.FieldCount())
		require.Equal(6, key.FieldCount())

		_, err := app.Build()
		require.NoError(err)
	})
}

func TestViewValidate(t *testing.T) {
	require := require.New(t)

	app := New()
	viewName := NewQName("test", "view")
	v := app.AddView(viewName)
	require.NotNil(v)

	t.Run("must be error if no pkey fields", func(t *testing.T) {
		_, err := app.Build()
		require.ErrorIs(err, ErrFieldsMissed)
	})

	v.AddPartField("pk1", DataKind_bool)

	t.Run("must be error if no ccols fields", func(t *testing.T) {
		_, err := app.Build()
		require.ErrorIs(err, ErrFieldsMissed)
	})

	v.AddClustColumn("cc1", DataKind_string)
	_, err := app.Build()
	require.NoError(err)

	t.Run("must be error if there a variable length field is not last in ccols", func(t *testing.T) {
		v.AddClustColumn("cc2", DataKind_int64)
		_, err := app.Build()
		require.ErrorIs(err, ErrInvalidDataKind)
		require.ErrorContains(err, "cc1")
	})
}
