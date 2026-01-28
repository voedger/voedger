/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts_test

import (
	"context"
	"fmt"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/appdef/filter"
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

func ExampleIAppPartition_IsLimitExceeded() {
	cmd1Name := appdef.NewQName("test", "cmd1")
	cmd2Name := appdef.NewQName("test", "cmd2")
	adb, app := func() (appdef.IAppDefBuilder, appdef.IAppDef) {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsName := appdef.NewQName("test", "workspace")
		wsb := adb.AddWorkspace(wsName)
		wsb.AddCDoc(appdef.NewQName("test", "WSDesc"))
		wsb.SetDescriptor(appdef.NewQName("test", "WSDesc"))
		_ = wsb.AddCommand(cmd1Name)
		_ = wsb.AddCommand(cmd2Name)

		// Add rate and limit for workspace: 5 commands per minute
		wsRateName := appdef.NewQName("test", "wsRate")
		wsb.AddRate(wsRateName, 5, time.Minute, []appdef.RateScope{appdef.RateScope_Workspace})
		wsb.AddLimit(
			appdef.NewQName("test", "wsLimit"),
			[]appdef.OperationKind{appdef.OperationKind_Execute},
			appdef.LimitFilterOption_ALL,
			filter.AllWSFunctions(wsName),
			wsRateName)

		// Add rate and limit for IP: 3 commands per minute for each IP address for each command
		ipRateName := appdef.NewQName("test", "ipRate")
		wsb.AddRate(ipRateName, 3, time.Minute, []appdef.RateScope{appdef.RateScope_IP})
		wsb.AddLimit(
			appdef.NewQName("test", "ipLimit"),
			[]appdef.OperationKind{appdef.OperationKind_Execute},
			appdef.LimitFilterOption_EACH,
			filter.AllWSFunctions(wsName),
			ipRateName)

		return adb, adb.MustBuild()
	}()

	appConfigs := istructsmem.AppConfigsType{}
	appConfigs.AddBuiltInAppConfig(istructs.AppQName_test1_app1, adb).SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)

	appStructsProvider := istructsmem.Provide(
		appConfigs,
		iratesce.TestBucketsFactory,
		payloads.ProvideIAppTokensFactory(itokensjwt.TestTokensJWT()),
		provider.Provide(mem.Provide(testingu.MockTime), ""), isequencer.SequencesTrustLevel_0, nil)

	vvmCtx, cancel := context.WithCancel(context.Background())
	appParts, cleanup, err := appparts.New2(
		vvmCtx,
		appStructsProvider,
		appparts.NullSyncActualizerFactory,
		appparts.NullActualizerRunner,
		appparts.NullSchedulerRunner,
		appparts.NullExtensionEngineFactories,
		iratesce.TestBucketsFactory,
	)
	if err != nil {
		panic(err)
	}
	defer func() {
		cancel()
		cleanup()
	}()
	appParts.DeployApp(istructs.AppQName_test1_app1, nil, app, 1, appparts.PoolSize(1, 1, 1, 1), istructs.DefaultNumAppWorkspaces)
	appParts.DeployAppPartitions(istructs.AppQName_test1_app1, []istructs.PartitionID{1})

	partition, err := appParts.Borrow(istructs.AppQName_test1_app1, 1, appparts.ProcessorKind_Command)
	if err != nil {
		panic(err)
	}
	defer partition.Release()

	// Check limits for cmd1
	fmt.Println(partition.IsLimitExceeded(cmd1Name, appdef.OperationKind_Execute, 1, `addr1`))
	fmt.Println(partition.IsLimitExceeded(cmd1Name, appdef.OperationKind_Execute, 1, `addr1`))
	fmt.Println(partition.IsLimitExceeded(cmd1Name, appdef.OperationKind_Execute, 1, `addr1`))

	fmt.Println(partition.IsLimitExceeded(cmd1Name, appdef.OperationKind_Execute, 1, `addr1`)) // Exceeded IP

	// Check limits for cmd2
	fmt.Println(partition.IsLimitExceeded(cmd2Name, appdef.OperationKind_Execute, 1, `addr1`))
	fmt.Println(partition.IsLimitExceeded(cmd2Name, appdef.OperationKind_Execute, 1, `addr1`))

	fmt.Println(partition.IsLimitExceeded(cmd2Name, appdef.OperationKind_Execute, 1, `addr1`)) // Exceeded WS

	testingu.MockTime.Add(time.Minute) // Wait for the next minute

	// check limits is respawned
	fmt.Println(partition.IsLimitExceeded(cmd1Name, appdef.OperationKind_Execute, 1, `addr1`))
	fmt.Println(partition.IsLimitExceeded(cmd2Name, appdef.OperationKind_Execute, 1, `addr1`))

	// check limits for unlimited command
	fmt.Println(partition.IsLimitExceeded(appdef.NewQName("test", "cmd3"), appdef.OperationKind_Execute, 1, `addr1`))
	fmt.Println(partition.IsLimitExceeded(appdef.NewQName("test", "cmd3"), appdef.OperationKind_Execute, 1, `addr1`))
	fmt.Println(partition.IsLimitExceeded(appdef.NewQName("test", "cmd3"), appdef.OperationKind_Execute, 1, `addr1`))
	fmt.Println(partition.IsLimitExceeded(appdef.NewQName("test", "cmd3"), appdef.OperationKind_Execute, 1, `addr1`))

	// Output:
	// false .
	// false .
	// false .
	// true test.ipLimit
	// false .
	// false .
	// true test.wsLimit
	// false .
	// false .
	// false .
	// false .
	// false .
	// false .
}
