/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 */

package cluster

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/goutils/testingu"
	"github.com/voedger/voedger/pkg/istorage/mem"
	"github.com/voedger/voedger/pkg/istorage/provider"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/itokensjwt"
	"github.com/voedger/voedger/pkg/sys/uniques"
)

func TestValidateAppDeployment(t *testing.T) {
	for _, test := range []struct {
		name       string
		absent     bool
		partitions istructs.NumAppPartitions
		workspaces istructs.NumAppWorkspaces
		message    string
	}{
		{name: "compatible", partitions: 3, workspaces: 4},
		{name: "absent", absent: true},
		{name: "partition count changed", partitions: 2, workspaces: 4,
			message: "num partitions changed: app test1/app1 declaring NumPartitions=3 but was previously deployed with NumPartitions=2"},
		{name: "workspace count changed", partitions: 3, workspaces: 2,
			message: "num application workspaces changed: app test1/app1 declaring NumAppWorkspaces=4 but was previously deployed with NumAppWorkspaces=2"},
	} {
		t.Run(test.name, func(t *testing.T) {
			adb := builder.New()
			adb.AddPackage(ClusterPackage, ClusterPackageFQN)
			ws := adb.AddWorkspace(appdef.NewQName(ClusterPackage, "workspace"))
			descriptor := appdef.NewQName(ClusterPackage, "descriptor")
			ws.AddCDoc(descriptor)
			ws.SetDescriptor(descriptor)
			app := ws.AddWDoc(qNameWDocApp)
			app.AddField(Field_AppQName, appdef.DataKind_string, true)
			app.AddField(Field_NumPartitions, appdef.DataKind_int32, true)
			app.AddField(Field_NumAppWorkspaces, appdef.DataKind_int32, true)
			uniqueQName := appdef.NewQName(ClusterPackage, "uniqueAppQName")
			app.AddUnique(uniqueQName, []appdef.FieldName{Field_AppQName})
			uniquesQName := appdef.NewQName(appdef.SysPackage, "Uniques")
			view := ws.AddView(uniquesQName)
			view.Key().PartKey().AddField("QName", appdef.DataKind_QName).AddField("ValuesHash", appdef.DataKind_int64)
			view.Key().ClustCols().AddField("Values", appdef.DataKind_bytes)
			view.Value().AddRefField("ID", false)

			configs := make(istructsmem.AppConfigsType)
			cfg := configs.AddBuiltInAppConfig(istructs.AppQName_sys_cluster, adb)
			cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
			storage := provider.Provide(mem.Provide(testingu.MockTime))
			t.Cleanup(storage.Stop)
			tokens := payloads.ProvideIAppTokensFactory(itokensjwt.TestTokensJWT())
			structs, err := istructsmem.Provide(configs, tokens, storage, istructs.SequencesTrustLevel_0, nil).
				BuiltIn(istructs.AppQName_sys_cluster)
			require.NoError(t, err)

			if !test.absent {
				const recordID = istructs.FirstUserRecordID
				require.NoError(t, structs.Records().PutJSON(1, map[appdef.FieldName]any{
					appdef.SystemField_QName: qNameWDocApp.String(),
					appdef.SystemField_ID:    recordID,
					Field_AppQName:           istructs.AppQName_test1_app1.String(),
					Field_NumPartitions:      int32(test.partitions),
					Field_NumAppWorkspaces:   int32(test.workspaces),
				}))
				unique := appdef.WDoc(structs.AppDef().Type, qNameWDocApp).Uniques()[uniqueQName]
				values, err := uniques.GetUniqueKeyValuesFromMap(map[string]interface{}{
					Field_AppQName: istructs.AppQName_test1_app1.String(),
				}, unique.Fields(), uniqueQName)
				require.NoError(t, err)
				key := structs.ViewRecords().KeyBuilder(uniquesQName)
				key.PutQName("QName", uniqueQName)
				key.PutInt64("ValuesHash", coreutils.HashBytes(values))
				key.PutBytes("Values", values)
				value := structs.ViewRecords().NewValueBuilder(uniquesQName)
				value.PutRecordID("ID", recordID)
				require.NoError(t, structs.ViewRecords().Put(1, key, value))
			}

			exists, err := validateAppDeployment(structs, 1, istructs.AppQName_test1_app1, 3, 4)

			if test.message != "" {
				var responseError coreutils.SysError
				require.ErrorAs(t, err, &responseError)
				require.Equal(t, http.StatusConflict, responseError.HTTPStatus)
				require.Equal(t, test.message, responseError.Message)
			} else {
				require.NoError(t, err)
				require.Equal(t, !test.absent, exists)
			}
		})
	}
}
