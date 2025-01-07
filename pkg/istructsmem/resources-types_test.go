/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/iratesce"
	"github.com/voedger/voedger/pkg/istructs"
)

func Test_nullResource(t *testing.T) {
	require := require.New(t)

	n := newNullResource()
	require.Equal(istructs.ResourceKind_null, n.Kind())
	require.Equal(appdef.NullQName, n.QName())
}

func TestResourceEnumerator(t *testing.T) {
	require := require.New(t)

	var (
		cfg *AppConfigType
		app istructs.IAppStructs

		wsName       appdef.QName = appdef.NewQName("test", "workspace")
		cmdCreateDoc appdef.QName = appdef.NewQName("test", "CreateDoc")
		oDocName     appdef.QName = appdef.NewQName("test", "ODoc")

		cmdCreateObj         appdef.QName = appdef.NewQName("test", "CreateObj")
		cmdCreateObjUnlogged appdef.QName = appdef.NewQName("test", "CreateObjUnlogged")
		oObjName             appdef.QName = appdef.NewQName("test", "Object")

		cmdCUD appdef.QName = appdef.NewQName("test", "cudEvent")
	)

	t.Run("builds app", func(t *testing.T) {

		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		t.Run("must be ok to build application", func(t *testing.T) {
			doc := wsb.AddODoc(oDocName)
			doc.
				AddField("Int32", appdef.DataKind_int32, true).
				AddField("String", appdef.DataKind_string, false)

			obj := wsb.AddObject(oObjName)
			obj.
				AddField("Int32", appdef.DataKind_int32, true).
				AddField("String", appdef.DataKind_string, false)

			wsb.AddCommand(cmdCreateDoc).SetParam(oDocName)
			wsb.AddCommand(cmdCreateObj).SetParam(oObjName)
			wsb.AddCommand(cmdCreateObjUnlogged).SetUnloggedParam(oObjName)
			wsb.AddCommand(cmdCUD)
		})

		cfgs := make(AppConfigsType, 1)
		cfg = cfgs.AddBuiltInAppConfig(istructs.AppQName_test1_app1, adb)
		cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)

		cfg.Resources.Add(NewCommandFunction(cmdCreateDoc, NullCommandExec))
		cfg.Resources.Add(NewCommandFunction(cmdCreateObj, NullCommandExec))
		cfg.Resources.Add(NewCommandFunction(cmdCreateObjUnlogged, NullCommandExec))
		cfg.Resources.Add(NewCommandFunction(cmdCUD, NullCommandExec))

		provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), simpleStorageProvider())

		var err error
		app, err = provider.BuiltIn(istructs.AppQName_test1_app1)
		require.NoError(err)
	})

	t.Run("should be ok to enumerate all resources", func(t *testing.T) {
		cnt := 0
		for resName := range app.Resources().Resources {
			cnt++
			require.NotNil(app.Resources().QueryResource(resName))
		}
		require.EqualValues(4, cnt)
	})
}
