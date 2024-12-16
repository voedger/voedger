/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package schedulers_test

import (
	"context"
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts/internal/schedulers"
	"github.com/voedger/voedger/pkg/istructs"
)

func Example() {
	appName := istructs.AppQName_test1_app1
	partCnt := istructs.NumAppPartitions(2)
	wsCnt := istructs.NumAppWorkspaces(10)
	partID := istructs.PartitionID(1)

	ctx, stop := context.WithCancel(context.Background())

	schedulers := schedulers.New(appName, partCnt, wsCnt, partID)

	appDef := func(jobNames ...appdef.QName) appdef.IAppDef {
		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")
		wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))
		for _, name := range jobNames {
			wsb.AddJob(name).SetCronSchedule("@every 5s")
		}
		return adb.MustBuild()
	}

	run := func(ctx context.Context, _ appdef.AppQName, _ istructs.PartitionID, _ istructs.AppWorkspaceNumber, _ istructs.WSID, _ appdef.QName) {
		<-ctx.Done()
	}

	{
		// deploy partition with appDef version 1
		jobNames := appdef.MustParseQNames("test.j1", "test.j2")
		appDefV1 := appDef(jobNames...)

		schedulers.Deploy(ctx, appDefV1, run)

		fmt.Println(schedulers.Enum())
	}

	{
		// redeploy partition with appDef version 2
		jobNames := appdef.MustParseQNames("test.j2", "test.j3")
		appDefV2 := appDef(jobNames...)

		schedulers.Deploy(ctx, appDefV2, run)

		fmt.Println(schedulers.Enum())
	}

	{
		// stop vvm from context, wait schedulers finished
		stop()
		schedulers.Wait()
		fmt.Println(schedulers.Enum())
	}

	// Output:
	// map[test.j1:[140737488420865 140737488420867 140737488420869 140737488420871 140737488420873] test.j2:[140737488420865 140737488420867 140737488420869 140737488420871 140737488420873]]
	// map[test.j2:[140737488420865 140737488420867 140737488420869 140737488420871 140737488420873] test.j3:[140737488420865 140737488420867 140737488420869 140737488420871 140737488420873]]
	// map[]
}
