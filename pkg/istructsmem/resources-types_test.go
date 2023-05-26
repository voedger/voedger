/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
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

		cmdCreateDoc appdef.QName = appdef.NewQName("test", "CreateDoc")
		cDocName     appdef.QName = appdef.NewQName("test", "CDoc")

		cmdCreateObj         appdef.QName = appdef.NewQName("test", "CreateObj")
		cmdCreateObjUnlogged appdef.QName = appdef.NewQName("test", "CreateObjUnlogged")
		oObjName             appdef.QName = appdef.NewQName("test", "Object")

		cmdCUD appdef.QName = appdef.NewQName("test", "cudEvent")
	)

	t.Run("builds app", func(t *testing.T) {

		appDef := appdef.New()
		t.Run("must be ok to build application definition", func(t *testing.T) {
			cDocDef := appDef.AddCDoc(cDocName)
			cDocDef.
				AddField("Int32", appdef.DataKind_int32, true).
				AddField("String", appdef.DataKind_string, false)

			objDef := appDef.AddObject(oObjName)
			objDef.
				AddField("Int32", appdef.DataKind_int32, true).
				AddField("String", appdef.DataKind_string, false)
		})

		cfgs := make(AppConfigsType, 1)
		cfg = cfgs.AddConfig(istructs.AppQName_test1_app1, appDef)

		cfg.Resources.Add(NewCommandFunction(cmdCreateDoc, cDocName, appdef.NullQName, appdef.NullQName, NullCommandExec))
		cfg.Resources.Add(NewCommandFunction(cmdCreateObj, oObjName, appdef.NullQName, appdef.NullQName, NullCommandExec))
		cfg.Resources.Add(NewCommandFunction(cmdCreateObjUnlogged, appdef.NullQName, oObjName, appdef.NullQName, NullCommandExec))
		cfg.Resources.Add(NewCommandFunction(cmdCUD, appdef.NullQName, appdef.NullQName, appdef.NullQName, NullCommandExec))

		storage, err := simpleStorageProvider().AppStorage(istructs.AppQName_test1_app1)
		require.NoError(err)
		err = cfg.prepare(iratesce.TestBucketsFactory(), storage)
		require.NoError(err)

		provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), simpleStorageProvider())

		app, err = provider.AppStructs(istructs.AppQName_test1_app1)
		require.NoError(err)
	})

	t.Run("enumerate all resources", func(t *testing.T) {
		cnt := 0
		app.Resources().Resources(
			func(resName appdef.QName) {
				cnt++
				require.NotNil(app.Resources().QueryResource(resName))
			})

		require.EqualValues(4, cnt)
	})
}
