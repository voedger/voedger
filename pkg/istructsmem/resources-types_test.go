/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/iratesce"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/schemas"
)

func Test_nullResource(t *testing.T) {
	require := require.New(t)

	n := newNullResource()
	require.Equal(istructs.ResourceKind_null, n.Kind())
	require.Equal(schemas.NullQName, n.QName())
}

func TestResourceEnumerator(t *testing.T) {
	require := require.New(t)

	var (
		cfg *AppConfigType
		app istructs.IAppStructs

		cmdCreateDoc schemas.QName = schemas.NewQName("test", "CreateDoc")
		cDocName     schemas.QName = schemas.NewQName("test", "CDoc")

		cmdCreateObj         schemas.QName = schemas.NewQName("test", "CreateObj")
		cmdCreateObjUnlogged schemas.QName = schemas.NewQName("test", "CreateObjUnlogged")
		oObjName             schemas.QName = schemas.NewQName("test", "Object")

		cmdCUD schemas.QName = schemas.NewQName("test", "cudEvent")
	)

	t.Run("builds app", func(t *testing.T) {

		bld := schemas.NewSchemaCache()
		t.Run("must be ok to build schemas and resources", func(t *testing.T) {
			CDocSchema := bld.Add(cDocName, schemas.SchemaKind_CDoc)
			CDocSchema.
				AddField("Int32", schemas.DataKind_int32, true).
				AddField("String", schemas.DataKind_string, false)

			ObjSchema := bld.Add(oObjName, schemas.SchemaKind_Object)
			ObjSchema.
				AddField("Int32", schemas.DataKind_int32, true).
				AddField("String", schemas.DataKind_string, false)
		})

		cfgs := make(AppConfigsType, 1)
		cfg = cfgs.AddConfig(istructs.AppQName_test1_app1, bld)

		cfg.Resources.Add(NewCommandFunction(cmdCreateDoc, cDocName, schemas.NullQName, schemas.NullQName, NullCommandExec))
		cfg.Resources.Add(NewCommandFunction(cmdCreateObj, oObjName, schemas.NullQName, schemas.NullQName, NullCommandExec))
		cfg.Resources.Add(NewCommandFunction(cmdCreateObjUnlogged, schemas.NullQName, oObjName, schemas.NullQName, NullCommandExec))
		cfg.Resources.Add(NewCommandFunction(cmdCUD, schemas.NullQName, schemas.NullQName, schemas.NullQName, NullCommandExec))

		storage, err := simpleStorageProvder().AppStorage(istructs.AppQName_test1_app1)
		require.NoError(err)
		err = cfg.prepare(iratesce.TestBucketsFactory(), storage)
		require.NoError(err)

		provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), simpleStorageProvder())

		app, err = provider.AppStructs(istructs.AppQName_test1_app1)
		require.NoError(err)
	})

	t.Run("enumerate all resources", func(t *testing.T) {
		cnt := 0
		app.Resources().Resources(
			func(resName schemas.QName) {
				cnt++
				require.NotNil(app.Resources().QueryResource(resName))
			})

		require.EqualValues(4, cnt)
	})
}
