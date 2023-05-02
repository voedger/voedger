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
	viewName := NewQName("test", "view")
	view := app.AddView(viewName)
	require.NotNil(view)
	require.Equal(viewName, view.Name())

	def := view.Def()
	require.NotNil(def)
	require.Equal(viewName, def.QName())
	require.Equal(DefKind_ViewRecord, def.Kind())
	require.Equal(3, def.ContainerCount())

	pk := view.PartKeyDef()
	require.NotNil(pk)
	require.Equal(ViewPartitionKeyDefName(viewName), pk.QName())
	require.Equal(DefKind_ViewRecord_PartitionKey, pk.Kind())
	require.Zero(pk.ContainerCount())

	cc := view.ClustColsDef()
	require.NotNil(cc)
	require.Equal(ViewClusteringColumsDefName(viewName), cc.QName())
	require.Equal(DefKind_ViewRecord_ClusteringColumns, cc.Kind())
	require.Zero(cc.ContainerCount())

	val := view.ValueDef()
	require.NotNil(val)
	require.Equal(ViewValueDefName(viewName), val.QName())
	require.Equal(DefKind_ViewRecord_Value, val.Kind())
	require.Zero(val.ContainerCount())

	t.Run("must be ok to add partition key fields", func(t *testing.T) {
		view.AddPartField("pkF1", DataKind_int64)
		view.AddPartField("pkF2", DataKind_bool)
		require.Equal(2, pk.FieldCount())

		t.Run("panic if variable length field added to pk", func(t *testing.T) {
			require.Panics(func() {
				view.AddPartField("pkF3", DataKind_string)
			})
		})
	})

	t.Run("must be ok to add clustering columns fields", func(t *testing.T) {
		view.AddClustColumn("ccF1", DataKind_int64)
		view.AddClustColumn("ccF2", DataKind_QName)
		require.Equal(2, cc.FieldCount())

		t.Run("panic if field already exists in pk", func(t *testing.T) {
			require.Panics(func() {
				view.AddClustColumn("pkF1", DataKind_int64)
			})
		})
	})

	t.Run("must be ok to add value fields", func(t *testing.T) {
		view.AddValueField("valF1", DataKind_bool, true)
		view.AddValueField("valF2", DataKind_string, false)
		require.Equal(2+1, val.FieldCount()) // + sys.QName field
	})

	_, err := app.Build()
	require.NoError(err)

	t.Run("must be ok to add value fields to view after app build", func(t *testing.T) {
		view.AddValueField("valF3", DataKind_Event, false)
		require.Equal(3+1, val.FieldCount())

		_, err := app.Build()
		require.NoError(err)
	})

	t.Run("must be ok to add pk or cc fields to view after app build", func(t *testing.T) {
		view.AddPartField("pkF3", DataKind_QName)
		view.AddClustColumn("ccF3", DataKind_string)

		require.Equal(3, pk.FieldCount())
		require.Equal(3, cc.FieldCount())

		_, err := app.Build()
		require.NoError(err)
	})
}
