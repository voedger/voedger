/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package mock

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
)

func TestView(t *testing.T) {
	viewName := appdef.NewQName("test", "view")

	appDef := NewAppDef()

	view := NewView(viewName)
	view.
		AddPartField("pkFld", appdef.DataKind_int64).
		AddClustColumn("ccFld", appdef.DataKind_string).
		AddValueField("vFld1", appdef.DataKind_int64, true).
		AddValueField("vFld2", appdef.DataKind_string, false)

	appDef.AddView(view)

	t.Run("test result", func(t *testing.T) {
		require := require.New(t)

		require.Equal(4, appDef.DefCount())
		require.Equal(appDef.DefCount(), func() int {
			cnt := 0
			appDef.Defs(func(appdef.IDef) { cnt++ })
			return cnt
		}())

		t.Run("test view", func(t *testing.T) {
			view := appDef.Def(viewName)
			require.NotNil(view)
			require.Equal(viewName, view.QName())
			require.Equal(appdef.DefKind_ViewRecord, view.Kind())
			require.Equal(3, view.ContainerCount())

			t.Run("test partition key", func(t *testing.T) {
				c := view.Container(appdef.SystemContainer_ViewPartitionKey)
				require.NotNil(c)
				require.True(c.IsSys())
				require.Equal(appdef.ViewPartitionKeyDefName(view.QName()), c.Def())

				pk := appDef.Def(c.Def())
				require.NotNil(pk)
				require.Equal(appdef.DefKind_ViewRecord_PartitionKey, pk.Kind())
				require.Equal(view.ContainerDef(c.Name()), pk)

				require.Equal(1, pk.FieldCount())
				require.Equal(pk.FieldCount(), func() int {
					cnt := 0
					pk.Fields(func(f appdef.IField) {
						cnt++
						switch f.Name() {
						case "pkFld":
							require.Equal(appdef.DataKind_int64, f.DataKind())
							require.True(f.Required())
						default:
							require.Failf("unknown field «%v» in definition «%v»", f.Name(), pk.QName())
						}
					})
					return cnt
				}())
			})

			t.Run("test clustering columns", func(t *testing.T) {
				c := view.Container(appdef.SystemContainer_ViewClusteringCols)
				require.NotNil(c)
				require.True(c.IsSys())
				require.Equal(appdef.ViewClusteringColumsDefName(view.QName()), c.Def())

				cc := appDef.Def(c.Def())
				require.NotNil(cc)
				require.Equal(appdef.DefKind_ViewRecord_ClusteringColumns, cc.Kind())
				require.Equal(view.ContainerDef(c.Name()), cc)

				require.Equal(1, cc.FieldCount())
				require.Equal(cc.FieldCount(), func() int {
					cnt := 0
					cc.Fields(func(f appdef.IField) {
						cnt++
						switch f.Name() {
						case "ccFld":
							require.Equal(appdef.DataKind_string, f.DataKind())
							require.False(f.Required())
						default:
							require.Failf("unknown field «%v» in definition «%v»", f.Name(), cc.QName())
						}
					})
					return cnt
				}())
			})

			t.Run("test view value", func(t *testing.T) {
				c := view.Container(appdef.SystemContainer_ViewValue)
				require.NotNil(c)
				require.True(c.IsSys())
				require.Equal(appdef.ViewValueDefName(view.QName()), c.Def())

				v := appDef.Def(c.Def())
				require.NotNil(v)
				require.Equal(appdef.DefKind_ViewRecord_Value, v.Kind())
				require.Equal(view.ContainerDef(c.Name()), v)

				require.Equal(2, v.FieldCount())
				require.Equal(v.FieldCount(), func() int {
					cnt := 0
					v.Fields(func(f appdef.IField) {
						cnt++
						switch f.Name() {
						case "vFld1":
							require.Equal(appdef.DataKind_int64, f.DataKind())
							require.True(f.Required())
						case "vFld2":
							require.Equal(appdef.DataKind_string, f.DataKind())
							require.False(f.Required())
						default:
							require.Failf("unknown field «%v» in definition «%v»", f.Name(), v.QName())
						}
					})
					return cnt
				}())
			})
		})
	})
}
