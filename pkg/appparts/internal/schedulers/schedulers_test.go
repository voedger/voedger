/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package schedulers

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
	"github.com/voedger/voedger/pkg/istructs"
)

func TestSchedulersWaitTimeout(t *testing.T) {
	appName := istructs.AppQName_test1_app1
	partCnt := istructs.NumAppPartitions(2)
	wsCnt := istructs.NumAppWorkspaces(10)
	initialPartID := istructs.PartitionID(1)
	jobNames := appdef.MustParseQNames("test.j1", "test.j2")

	appDef := func() appdef.IAppDef {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")
		wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))
		for _, name := range jobNames {
			wsb.AddJob(name).SetCronSchedule("@every 5s")
		}
		return adb.MustBuild()
	}

	require := require.New(t)

	t.Run("should ok to wait for all actualizers finished", func(t *testing.T) {
		ctx, stop := context.WithCancel(context.Background())

		schedulers := New(appName, partCnt, wsCnt, initialPartID)

		app := appDef()

		runCalls := sync.Map{}
		runKey := func(j appdef.QName, ws istructs.AppWorkspaceNumber) string {
			return fmt.Sprintf("%s[%d]", j, ws)
		}
		for _, name := range jobNames {
			for ws := istructs.AppWorkspaceNumber(0); ws < istructs.AppWorkspaceNumber(wsCnt); ws++ {
				if ws%2 == 1 {
					runCalls.Store(runKey(name, ws), 1)
				}
			}
		}
		schedulers.Deploy(ctx, app,
			func(ctx context.Context, app appdef.AppQName, deployingPartID istructs.PartitionID, wsNum istructs.AppWorkspaceNumber, wsID istructs.WSID, name appdef.QName) {
				key := runKey(name, wsNum)
				require.True(runCalls.CompareAndDelete(key, 1), "scheduler %s was run more than once", key)

				require.Equal(appName, app)
				require.Equal(initialPartID, deployingPartID)
				require.Contains(jobNames, name)
				require.Equal(
					istructs.NewWSID(istructs.CurrentClusterID(), istructs.WSID(wsNum)+istructs.FirstBaseAppWSID),
					wsID,
					"wsID for %s", key,
				)

				<-ctx.Done()
			})

		require.Equal(
			map[appdef.QName][]istructs.WSID{
				jobNames[0]: {
					istructs.NewWSID(istructs.CurrentClusterID(), istructs.WSID(1)+istructs.FirstBaseAppWSID),
					istructs.NewWSID(istructs.CurrentClusterID(), istructs.WSID(3)+istructs.FirstBaseAppWSID),
					istructs.NewWSID(istructs.CurrentClusterID(), istructs.WSID(5)+istructs.FirstBaseAppWSID),
					istructs.NewWSID(istructs.CurrentClusterID(), istructs.WSID(7)+istructs.FirstBaseAppWSID),
					istructs.NewWSID(istructs.CurrentClusterID(), istructs.WSID(9)+istructs.FirstBaseAppWSID),
				},
				jobNames[1]: {
					istructs.NewWSID(istructs.CurrentClusterID(), istructs.WSID(1)+istructs.FirstBaseAppWSID),
					istructs.NewWSID(istructs.CurrentClusterID(), istructs.WSID(3)+istructs.FirstBaseAppWSID),
					istructs.NewWSID(istructs.CurrentClusterID(), istructs.WSID(5)+istructs.FirstBaseAppWSID),
					istructs.NewWSID(istructs.CurrentClusterID(), istructs.WSID(7)+istructs.FirstBaseAppWSID),
					istructs.NewWSID(istructs.CurrentClusterID(), istructs.WSID(9)+istructs.FirstBaseAppWSID),
				},
			},
			schedulers.Enum())

		// stop vvm from context
		stop()
		schedulers.Wait()

		runCalls.Range(func(key, value any) bool {
			require.Fail(fmt.Sprintf("scheduler %#v was not run", key))
			return true
		})
	})
}

func TestSchedulersDeploy(t *testing.T) {

	appName := istructs.AppQName_test1_app1
	const (
		partCnt = 2 // partition 0 should handle schedulers for single workspace 0, partition 1 should not
		wsCnt   = 1
	)
	jobName := appdef.MustParseQName("test.j1")

	app := func() appdef.IAppDef {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")
		adb.AddWorkspace(appdef.NewQName("test", "workspace")).
			AddJob(jobName).SetCronSchedule("@every 5s")
		return adb.MustBuild()
	}()

	require := require.New(t)

	ctx, stop := context.WithCancel(context.Background())

	var schedulers [partCnt]*PartitionSchedulers

	t.Run("should be ok to deploy if partitions does not handle schedulers", func(t *testing.T) {
		deploy := func(partID istructs.PartitionID) *PartitionSchedulers {
			s := New(appName, partCnt, wsCnt, partID)
			s.Deploy(ctx, app,
				func(ctx context.Context, _ appdef.AppQName, _ istructs.PartitionID, _ istructs.AppWorkspaceNumber, _ istructs.WSID, name appdef.QName) {
					<-ctx.Done()
				})
			return s
		}

		for pid := istructs.PartitionID(0); pid < partCnt; pid++ {
			schedulers[pid] = deploy(pid)
		}

		require.Equal(
			map[appdef.QName][]istructs.WSID{
				jobName: {
					istructs.NewWSID(istructs.CurrentClusterID(), istructs.WSID(0)+istructs.FirstBaseAppWSID),
				},
			},
			schedulers[0].Enum())

		require.Empty(schedulers[1].Enum(), "partition 1 should not handle schedulers")
	})

	t.Run("should be ok to wait for all actualizers finished", func(t *testing.T) {
		// stop vvm from context, wait actualizers finished
		stop()

		wg := sync.WaitGroup{}
		for pid := istructs.PartitionID(0); pid < partCnt; pid++ {
			wg.Add(1)
			go func(pid istructs.PartitionID) {
				defer wg.Done()
				schedulers[pid].Wait()
			}(pid)
		}

		wg.Wait()

		for pid := istructs.PartitionID(0); pid < partCnt; pid++ {
			require.Empty(schedulers[pid].Enum())
		}
	})
}
