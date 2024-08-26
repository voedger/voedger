/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package schedulers_test

import (
	"context"
	"fmt"
	"sync"

	"github.com/stretchr/testify/mock"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts/internal/schedulers"
	"github.com/voedger/voedger/pkg/istructs"
)

type mockSchedulerRunner struct {
	mock.Mock
	wg sync.WaitGroup
}

func (t *mockSchedulerRunner) Run(ctx context.Context, app appdef.AppQName, partID istructs.PartitionID, wsIdx int, wsID istructs.WSID, job appdef.QName) {
	t.wg.Add(1)
	defer t.wg.Done()

	t.Called(ctx, app, partID, wsIdx, wsID, job)

	<-ctx.Done()
}

func (t *mockSchedulerRunner) wait() {
	// the context should be stopped. Here we just wait for scheduler to finish
	t.wg.Wait()
}

func Example() {
	appName := istructs.AppQName_test1_app1
	partCnt := istructs.NumAppPartitions(2)
	wsCnt := istructs.NumAppWorkspaces(10)
	partID := istructs.PartitionID(1)

	ctx, stop := context.WithCancel(context.Background())

	runner := &mockSchedulerRunner{}

	schedulers := schedulers.New(appName, partCnt, wsCnt, partID)

	appDef := func(jobNames ...appdef.QName) appdef.IAppDef {
		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")
		for _, name := range jobNames {
			adb.AddJob(name).SetCronSchedule("@every 5s")
		}
		return adb.MustBuild()
	}

	{
		// deploy partition with appDef version 1
		jobNames := appdef.MustParseQNames("test.j1", "test.j2")
		appDefV1 := appDef(jobNames...)

		for ws := 0; ws < int(wsCnt); ws++ {
			for _, name := range jobNames {
				if ws%int(partCnt) == int(partID) {
					runner.On("Run", mock.Anything,
						appName,
						partID,
						ws,
						istructs.NewWSID(istructs.CurrentClusterID(), istructs.FirstBaseAppWSID+istructs.WSID(ws)),
						name).Once()
				}
			}
		}

		schedulers.Deploy(ctx, appDefV1, runner.Run)

		fmt.Println(schedulers.Enum())
	}

	{
		// redeploy partition with appDef version 2
		jobNames := appdef.MustParseQNames("test.j2", "test.j3")
		appDefV2 := appDef(jobNames...)

		for ws := 0; ws < int(wsCnt); ws++ {
			for _, name := range jobNames {
				if ws%int(partCnt) == int(partID) {
					runner.On("Run", mock.Anything,
						appName,
						partID,
						ws,
						istructs.NewWSID(istructs.CurrentClusterID(), istructs.FirstBaseAppWSID+istructs.WSID(ws)),
						name).Once()
				}
			}
		}

		schedulers.Deploy(ctx, appDefV2, runner.Run)

		fmt.Println(schedulers.Enum())
	}

	{
		// stop vvm from context, wait schedulers finished
		stop()

		runner.wait()

		fmt.Println(schedulers.Enum())
	}

	// Output:
	// map[test.j1:[140737488420865 140737488420867 140737488420869 140737488420871 140737488420873] test.j2:[140737488420865 140737488420867 140737488420869 140737488420871 140737488420873]]
	// map[test.j2:[140737488420865 140737488420867 140737488420869 140737488420871 140737488420873] test.j3:[140737488420865 140737488420867 140737488420869 140737488420871 140737488420873]]
	// map[]
}
