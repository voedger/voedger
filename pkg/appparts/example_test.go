/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts_test

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/goutils/testingu"
	"github.com/voedger/voedger/pkg/iratesce"
	"github.com/voedger/voedger/pkg/isequencer"
	"github.com/voedger/voedger/pkg/istorage/mem"
	"github.com/voedger/voedger/pkg/istorage/provider"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/itokensjwt"
)

func Example() {
	wsName := appdef.NewQName("test", "workspace")
	verInfo := appdef.NewQName("test", "verInfo")
	buildAppDef := func(ver ...string) (appdef.IAppDefBuilder, appdef.IAppDef) {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		ws := adb.AddWorkspace(wsName)
		ws.AddCDoc(verInfo).SetComment(ver...)
		ws.SetDescriptor(verInfo)

		app, err := adb.Build()
		if err != nil {
			panic(err)
		}
		return adb, app
	}

	appConfigs := istructsmem.AppConfigsType{}
	adb_1_v1, app_1_v1 := buildAppDef("app-1 ver.1")
	adb_2_v1, app_2_v1 := buildAppDef("app-2 ver.1")
	appConfigs.AddBuiltInAppConfig(istructs.AppQName_test1_app1, adb_1_v1).SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
	appConfigs.AddBuiltInAppConfig(istructs.AppQName_test1_app2, adb_2_v1).SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)

	appStructsProvider := istructsmem.Provide(
		appConfigs,
		iratesce.TestBucketsFactory,
		payloads.ProvideIAppTokensFactory(itokensjwt.TestTokensJWT()),
		provider.Provide(mem.Provide(testingu.MockTime), ""), isequencer.SequencesTrustLevel_0, nil)

	appParts, cleanupParts := appparts.NewTestAppParts(appStructsProvider)
	defer cleanupParts()

	report := func(part appparts.IAppPartition) {
		fmt.Println(part.App(), "partition", part.ID())
		ver := appdef.CDoc(part.AppStructs().AppDef().Type, verInfo)
		fmt.Println("-", ver, ver.Comment())
	}

	fmt.Println("*** Add ver 1 ***")

	appParts.DeployApp(istructs.AppQName_test1_app1, nil, app_1_v1, 1, appparts.PoolSize(2, 2, 2, 2), istructs.DefaultNumAppWorkspaces)
	appParts.DeployApp(istructs.AppQName_test1_app2, nil, app_2_v1, 1, appparts.PoolSize(2, 2, 2, 2), istructs.DefaultNumAppWorkspaces)

	appParts.DeployAppPartitions(istructs.AppQName_test1_app1, []istructs.PartitionID{1})
	appParts.DeployAppPartitions(istructs.AppQName_test1_app2, []istructs.PartitionID{1})

	a1_v1_p1, err := appParts.Borrow(istructs.AppQName_test1_app1, 1, appparts.ProcessorKind_Command)
	if err != nil {
		panic(err)
	}
	defer a1_v1_p1.Release()

	report(a1_v1_p1)

	a2_v1_p1, err := appParts.Borrow(istructs.AppQName_test1_app2, 1, appparts.ProcessorKind_Query)
	if err != nil {
		panic(err)
	}
	defer a2_v1_p1.Release()

	report(a2_v1_p1)

	// Output:
	// *** Add ver 1 ***
	// test1/app1 partition 1
	// - CDoc «test.verInfo» app-1 ver.1
	// test1/app2 partition 1
	// - CDoc «test.verInfo» app-2 ver.1
}
