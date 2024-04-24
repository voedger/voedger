/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package sys_it

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/apps/sys/clusterapp"
	"github.com/voedger/voedger/pkg/btstrp"
	"github.com/voedger/voedger/pkg/cluster"
	"github.com/voedger/voedger/pkg/istructs"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/sys/authnz"
	coreutils "github.com/voedger/voedger/pkg/utils"
	it "github.com/voedger/voedger/pkg/vit"
	"github.com/voedger/voedger/pkg/vvm"
)

func TestAppsProtection(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	// try to deploy with different NumPartitions
	var clusterBuiltinApp btstrp.ClusterBuiltInApp
	otherApps := []appparts.BuiltInApp{}
	for _, app := range vit.BuiltInAppsPackages {
		if app.Name == istructs.AppQName_sys_cluster {
			clusterBuiltinApp = btstrp.ClusterBuiltInApp(app.BuiltInApp)
		} else {
			otherApps = append(otherApps, app.BuiltInApp)
		}
	}

	err := btstrp.Bootstrap(vit.IFederation, vit.IAppStructsProvider, vit.TimeFunc, vit.IAppPartitions, clusterBuiltinApp, otherApps, vit.ITokens)
	require.NoError(err)
}

func TestDeployAppErrors(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	sysToken, err := payloads.GetSystemPrincipalToken(vit.ITokens, istructs.AppQName_sys_cluster)
	require.NoError(err)

	t.Run("sys/cluster can not be deployed by c.cluster.DeployApp", func(t *testing.T) {
		body := fmt.Sprintf(`{"args":{"AppQName":"%s","NumPartitions":1}}`, istructs.AppQName_sys_cluster)
		vit.PostApp(istructs.AppQName_sys_cluster, clusterapp.ClusterAppPseudoWSID, "c.cluster.DeployApp", body,
			coreutils.WithAuthorizeBy(sysToken), coreutils.Expect400()).Println()
	})

	t.Run("409 conflict on deploy already deployed", func(t *testing.T) {
		body := fmt.Sprintf(`{"args":{"AppQName":"%s","NumPartitions":1}}`, istructs.AppQName_test1_app1)
		resp := vit.PostApp(istructs.AppQName_sys_cluster, clusterapp.ClusterAppPseudoWSID, "c.cluster.DeployApp", body,
			coreutils.WithAuthorizeBy(sysToken))

		// check nothing is made
		require.Empty(resp.NewIDs)
		checkCDocsWSDesc(vit.VVMConfig, vit.VVM, require)
	})
}

func TestAppWSInitIndempotency(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	checkCDocsWSDesc(vit.VVMConfig, vit.VVM, require)

	for _, app := range vit.BuiltInAppsPackages {
		as, err := vit.AppStructs(app.Name)
		require.NoError(err)
		initedWSIDs, err := cluster.InitAppWSes(as, as.NumAppWorkspaces(), app.NumParts, istructs.UnixMilli(vit.TimeFunc().UnixMilli()))
		require.NoError(err)
		require.Empty(initedWSIDs)
	}
}

func checkCDocsWSDesc(vvmCfg *vvm.VVMConfig, vvm *vvm.VVM, require *require.Assertions) {
	for appQName := range vvmCfg.VVMAppsBuilder {
		as, err := vvm.AppStructs(appQName)
		require.NoError(err)
		for wsNum := 0; istructs.NumAppWorkspaces(wsNum) < as.NumAppWorkspaces(); wsNum++ {
			appWSID := istructs.NewWSID(istructs.MainClusterID, istructs.WSID(wsNum+int(istructs.FirstBaseAppWSID)))
			existingCDocWSDesc, err := as.Records().GetSingleton(appWSID, authnz.QNameCDocWorkspaceDescriptor)
			require.NoError(err)
			require.Equal(authnz.QNameCDocWorkspaceDescriptor, existingCDocWSDesc.QName())
		}
	}
}
