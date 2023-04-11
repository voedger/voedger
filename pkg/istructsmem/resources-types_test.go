/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/untillpro/voedger/pkg/iratesce"
	"github.com/untillpro/voedger/pkg/istructs"
	"github.com/untillpro/voedger/pkg/schemas"
)

func Test_nullResource(t *testing.T) {
	require := require.New(t)

	n := newNullResource()
	require.Equal(istructs.ResourceKind_null, n.Kind())
	require.Equal(istructs.NullQName, n.QName())
}

func TestResourceEnumerator(t *testing.T) {
	require := require.New(t)

	var (
		cfg *AppConfigType
		app istructs.IAppStructs

		cmdCreateDoc istructs.QName = istructs.NewQName("test", "CreateDoc")
		cDocName     istructs.QName = istructs.NewQName("test", "CDoc")

		cmdCreateObj         istructs.QName = istructs.NewQName("test", "CreateObj")
		cmdCreateObjUnlogged istructs.QName = istructs.NewQName("test", "CreateObjUnlogged")
		oObjName             istructs.QName = istructs.NewQName("test", "Object")

		cmdCUD istructs.QName = istructs.NewQName("test", "cudEvent")
	)

	t.Run("builds app", func(t *testing.T) {

		schemas := schemas.NewSchemaCache()
		t.Run("must be ok to build schemas and resources", func(t *testing.T) {
			CDocSchema := schemas.Add(cDocName, istructs.SchemaKind_CDoc)
			CDocSchema.
				AddField("Int32", istructs.DataKind_int32, true).
				AddField("String", istructs.DataKind_string, false)

			ObjSchema := schemas.Add(oObjName, istructs.SchemaKind_Object)
			ObjSchema.
				AddField("Int32", istructs.DataKind_int32, true).
				AddField("String", istructs.DataKind_string, false)

			require.NoError(schemas.ValidateSchemas())
		})

		cfgs := make(AppConfigsType, 1)
		cfg = cfgs.AddConfig(istructs.AppQName_test1_app1, schemas)

		cfg.Resources.Add(NewCommandFunction(cmdCreateDoc, cDocName, istructs.NullQName, istructs.NullQName, NullCommandExec))
		cfg.Resources.Add(NewCommandFunction(cmdCreateObj, oObjName, istructs.NullQName, istructs.NullQName, NullCommandExec))
		cfg.Resources.Add(NewCommandFunction(cmdCreateObjUnlogged, istructs.NullQName, oObjName, istructs.NullQName, NullCommandExec))
		cfg.Resources.Add(NewCommandFunction(cmdCUD, istructs.NullQName, istructs.NullQName, istructs.NullQName, NullCommandExec))

		storage, err := simpleStorageProvder().AppStorage(istructs.AppQName_test1_app1)
		require.NoError(err)
		err = cfg.prepare(iratesce.TestBucketsFactory(), storage)
		require.NoError(err)

		provider, err := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), simpleStorageProvder())
		require.NoError(err)

		app, err = provider.AppStructs(istructs.AppQName_test1_app1)
		require.NoError(err)
	})

	t.Run("enumerate all resources", func(t *testing.T) {
		cnt := 0
		app.Resources().Resources(
			func(resName istructs.QName) {
				cnt++
				require.NotNil(app.Resources().QueryResource(resName))
			})

		require.EqualValues(4, cnt)
	})
}
