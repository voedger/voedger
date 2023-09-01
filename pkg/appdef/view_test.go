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

	ab := New()

	objName := NewQName("test", "object")
	_ = ab.AddObject(objName)

	viewName := NewQName("test", "view")
	vb := ab.AddView(viewName)

	t.Run("must be ok to build view", func(t *testing.T) {

		t.Run("must be ok to add partition key fields", func(t *testing.T) {
			vb.AddPartField("pkF1", DataKind_int64)
			vb.AddPartField("pkF2", DataKind_bool)

			t.Run("panic if variable length field added to pk", func(t *testing.T) {
				require.Panics(func() {
					vb.AddPartField("pkF3", DataKind_string)
				})
			})
		})

		t.Run("must be ok to add clustering columns fields", func(t *testing.T) {
			vb.AddClustColumn("ccF1", DataKind_int64)
			vb.AddClustColumn("ccF2", DataKind_QName)

			t.Run("panic if field already exists in pk", func(t *testing.T) {
				require.Panics(func() {
					vb.AddClustColumn("pkF1", DataKind_int64)
				})
			})
		})

		t.Run("must be ok to add value fields", func(t *testing.T) {
			vb.AddValueField("valF1", DataKind_bool, true)
			vb.AddValueField("valF2", DataKind_string, false)
		})
	})

	app, err := ab.Build()
	require.NoError(err)
	view := app.View(viewName)

	t.Run("must be ok to read view", func(t *testing.T) {
		require.Equal(viewName, view.QName())
		require.Equal(DefKind_ViewRecord, view.Kind())
		require.Equal(2, view.ContainerCount()) // key + value

		t.Run("must be ok to read view full key", func(t *testing.T) {
			key := view.Key()
			require.Equal(view.Container(SystemContainer_ViewKey).Def(), key)
			require.Equal(ViewKeyDefName(viewName), key.QName())
			require.Equal(DefKind_ViewRecord_Key, key.Kind())
			require.Equal(2, key.ContainerCount()) // pk + cc
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

		t.Run("must be ok to read view partition key", func(t *testing.T) {
			pk := view.Key().Partition()
			require.Equal(view.Key().Container(SystemContainer_ViewPartitionKey).Def(), pk)
			require.Equal(ViewPartitionKeyDefName(viewName), pk.QName())
			require.Equal(DefKind_ViewRecord_PartitionKey, pk.Kind())
			require.Equal(2, pk.FieldCount())
		})

		t.Run("must be ok to read view clustering columns", func(t *testing.T) {
			cc := view.Key().ClustCols()
			require.Equal(view.Key().Container(SystemContainer_ViewClusteringCols).Def(), cc)
			require.Equal(ViewClusteringColumnsDefName(viewName), cc.QName())
			require.Equal(DefKind_ViewRecord_ClusteringColumns, cc.Kind())
			require.Equal(2, cc.FieldCount())
		})

		t.Run("must be ok to read view value", func(t *testing.T) {
			val := view.Value()
			require.Equal(ViewValueDefName(viewName), val.QName())
			require.Equal(DefKind_ViewRecord_Value, val.Kind())
			require.Equal(2, val.UserFieldCount())
		})

		t.Run("must be ok to cast Def() as IView", func(t *testing.T) {
			d := app.Def(viewName)
			require.NotNil(d)
			require.Equal(DefKind_ViewRecord, d.Kind())

			v, ok := d.(IView)
			require.True(ok)
			require.Equal(v, view)

			k, ok := app.Def(ViewKeyDefName(viewName)).(IViewKey)
			require.True(ok)
			require.Equal(k, v.Key())
		})

		require.Nil(ab.View(NewQName("test", "unknown")), "find unknown view must return nil")

		t.Run("must be nil if not view", func(t *testing.T) {
			require.Nil(app.View(objName))

			d := app.Def(objName)
			require.NotNil(d)
			v, ok := d.(IView)
			require.False(ok)
			require.Nil(v)
		})
	})

	t.Run("must be ok to add fields to view after app build", func(t *testing.T) {
		vb.AddPartField("pkF3", DataKind_QName)
		vb.AddClustColumn("ccF3", DataKind_string)
		vb.AddValueField("valF3", DataKind_Event, false)

		_, err := ab.Build()
		require.NoError(err)

		require.Equal(3, view.Key().Partition().FieldCount())
		require.Equal(3, view.Key().ClustCols().FieldCount())
		require.Equal(6, view.Key().FieldCount())
		require.Equal(3, view.Value().UserFieldCount())
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
