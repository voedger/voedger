/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package sys_it

import (
	"fmt"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/apps"
	"github.com/voedger/voedger/pkg/apps/sys/clusterapp"
	"github.com/voedger/voedger/pkg/btstrp"
	"github.com/voedger/voedger/pkg/cluster"
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istorage/mem"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/parser"
	"github.com/voedger/voedger/pkg/sys"
	"github.com/voedger/voedger/pkg/sys/authnz"
	"github.com/voedger/voedger/pkg/sys/smtp"
	coreutils "github.com/voedger/voedger/pkg/utils"
	it "github.com/voedger/voedger/pkg/vit"
	"github.com/voedger/voedger/pkg/vvm"
)

func TestAppsDeploymentDescriptorProtection(t *testing.T) {
	require := require.New(t)
	memStorage := mem.Provide()
	keyspacePrefix := t.Name()

	// launch the VVM with an app with a certain NumParts and NumAppWorkspaces
	numParts := istructs.NumAppPartitions(42)
	numAppWS := istructs.NumAppWorkspaces(43)
	cfg := getTestCfg(numParts, numAppWS, memStorage, keyspacePrefix)
	vit := it.NewVIT(t, &cfg)

	// try to launch the VVM with the app with NumParts that differs from the previously deployed one
	var clusterApp btstrp.ClusterBuiltInApp
	otherApps := []appparts.BuiltInApp{}
	for _, app := range vit.BuiltInAppsPackages {
		if app.Name == istructs.AppQName_sys_cluster {
			clusterApp = btstrp.ClusterBuiltInApp(app.BuiltInApp)
		} else {
			otherApps = append(otherApps, app.BuiltInApp)
		}
	}

	t.Run("fail on NumPartitions change", func(t *testing.T) {
		appParts, cleanup, err := appparts.New(vit.IAppStructsProvider)
		require.NoError(err)
		defer cleanup()
		otherApps[0].AppDeploymentDescriptor.NumParts++
		defer func() {
			otherApps[0].AppDeploymentDescriptor.NumParts--
		}()
		err = btstrp.Bootstrap(vit.IFederation, vit.IAppStructsProvider, vit.TimeFunc, appParts, clusterApp, otherApps, vit.ITokens)
		require.ErrorIs(err, btstrp.ErrNumPartitionsChanged)
	})

	t.Run("fail on NumAppPartitions change", func(t *testing.T) {
		appParts, cleanup, err := appparts.New(vit.IAppStructsProvider)
		require.NoError(err)
		defer cleanup()
		otherApps[0].AppDeploymentDescriptor.NumAppWorkspaces++
		defer func() {
			otherApps[0].AppDeploymentDescriptor.NumAppWorkspaces--
		}()
		err = btstrp.Bootstrap(vit.IFederation, vit.IAppStructsProvider, vit.TimeFunc, appParts, clusterApp, otherApps, vit.ITokens)
		require.ErrorIs(err, btstrp.ErrNumAppWorkspacesChanged)
	})
}

func getTestCfg(numParts istructs.NumAppPartitions, numAppWS istructs.NumAppWorkspaces, storage istorage.IAppStorageFactory, testName string) it.VITConfig {
	fs := fstest.MapFS{
		"app.vsql": &fstest.MapFile{
			Data: []byte(`APPLICATION app1();`),
		},
	}
	app1PackageFS := parser.PackageFS{
		Path: it.App1PkgPath,
		FS:   fs,
	}
	return it.NewOwnVITConfig(
		it.WithApp(istructs.AppQName_test1_app1, func(apis apps.APIs, cfg *istructsmem.AppConfigType, ep extensionpoints.IExtensionPoint) apps.BuiltInAppDef {
			sysPkg := sys.Provide(cfg, smtp.Cfg{}, ep, nil, apis.TimeFunc, apis.ITokens, apis.IFederation, apis.IAppStructsProvider, apis.IAppTokensFactory,
				nil, apis.IAppStorageProvider)
			return apps.BuiltInAppDef{
				AppDeploymentDescriptor: appparts.AppDeploymentDescriptor{
					NumParts:         numParts,
					EnginePoolSize:   it.DefaultTestAppEnginesPool,
					NumAppWorkspaces: numAppWS,
				},
				AppQName: istructs.AppQName_test1_app1,
				Packages: []parser.PackageFS{sysPkg, app1PackageFS},
			}
		}),
		it.WithVVMConfig(func(cfg *vvm.VVMConfig) {
			// use predefined storage
			cfg.StorageFactory = func() (provider istorage.IAppStorageFactory, err error) {
				return storage, nil
			}
			cfg.KeyspaceNameSuffix = testName
		}),
	)
}

func TestDeployAppErrors(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	sysToken, err := payloads.GetSystemPrincipalToken(vit.ITokens, istructs.AppQName_sys_cluster)
	require.NoError(err)

	t.Run("sys/cluster can not be deployed by c.cluster.DeployApp", func(t *testing.T) {
		body := fmt.Sprintf(`{"args":{"AppQName":"%s","NumPartitions":1,"NumAppWorkspaces":1}}`, istructs.AppQName_sys_cluster)
		vit.PostApp(istructs.AppQName_sys_cluster, clusterapp.ClusterAppPseudoWSID, "c.cluster.DeployApp", body,
			coreutils.WithAuthorizeBy(sysToken), coreutils.Expect400()).Println()
	})

	t.Run("409 conflict on deploy already deployed", func(t *testing.T) {
		body := fmt.Sprintf(`{"args":{"AppQName":"%s","NumPartitions":1,"NumAppWorkspaces":1}}`, istructs.AppQName_test1_app1)
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

	// init app ws again (first is done on NewVIT()) -> expect no errors + assume next tests will work as well
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
