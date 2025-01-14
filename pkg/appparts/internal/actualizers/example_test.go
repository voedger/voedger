/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package actualizers_test

import (
	"context"
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/appdef/filter"
	"github.com/voedger/voedger/pkg/appparts/internal/actualizers"
	"github.com/voedger/voedger/pkg/istructs"
)

func Example() {
	appName := istructs.AppQName_test1_app1
	partID := istructs.PartitionID(1)

	ctx, stop := context.WithCancel(context.Background())

	actualizers := actualizers.New(appName, partID)

	appDef := func(prjNames ...appdef.QName) appdef.IAppDef {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")
		wsName := appdef.NewQName("test", "workspace")
		wsb := adb.AddWorkspace(wsName)
		_ = wsb.AddCommand(appdef.NewQName("test", "command"))
		for _, name := range prjNames {
			prj := wsb.AddProjector(name)
			prj.Events().Add(
				[]appdef.OperationKind{appdef.OperationKind_Execute},
				filter.AllWSFunctions(wsName))
			prj.SetSync(false)
		}
		return adb.MustBuild()
	}

	run := func(ctx context.Context, _ appdef.AppQName, _ istructs.PartitionID, _ appdef.QName) { <-ctx.Done() }

	{
		// deploy partition with appDef version 1
		prjNames := appdef.MustParseQNames("test.p1", "test.p2", "test.p3", "test.p4", "test.p5")
		appDefV1 := appDef(prjNames...)

		actualizers.Deploy(ctx, appDefV1, run)

		fmt.Println(actualizers.Enum())
	}

	{
		// redeploy partition with appDef version 2
		prjNames := appdef.MustParseQNames("test.p3", "test.p4", "test.p5", "test.p6", "test.p7")
		appDefV2 := appDef(prjNames...)

		actualizers.Deploy(ctx, appDefV2, run)

		fmt.Println(actualizers.Enum())
	}

	{
		// stop vvm from context, wait actualizers finished
		stop()
		actualizers.Wait()
		fmt.Println(actualizers.Enum())
	}

	// Output:
	// [test.p1 test.p2 test.p3 test.p4 test.p5]
	// [test.p3 test.p4 test.p5 test.p6 test.p7]
	// []
}
