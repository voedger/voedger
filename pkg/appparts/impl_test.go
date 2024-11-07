/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts/internal/schedulers"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
	"github.com/voedger/voedger/pkg/iratesce"
	"github.com/voedger/voedger/pkg/istorage/mem"
	"github.com/voedger/voedger/pkg/istorage/provider"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/itokensjwt"
)

type mockRunner struct {
	appParts IAppPartitions
}

func (mr *mockRunner) newAndRun(ctx context.Context, app appdef.AppQName, partID istructs.PartitionID, kind ProcessorKind) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// partition should be borrowed and released
			p, err := mr.appParts.WaitForBorrow(ctx, app, partID, kind)
			if err != nil {
				if errors.Is(err, ctx.Err()) {
					return // context canceled while wait for borrowed partition
				}
				panic(err) // unexpected error while waiting for borrowed partition
			}
			// simulate work, like p.Invoke(…)
			time.Sleep(time.Millisecond)
			p.Release()
		}
	}
}

func (mr *mockRunner) setAppPartitions(ap IAppPartitions) {
	mr.appParts = ap
}

type mockActualizerRunner struct {
	mock.Mock
	mockRunner
	IActualizerRunner
}

func (ar *mockActualizerRunner) NewAndRun(ctx context.Context, app appdef.AppQName, partID istructs.PartitionID, name appdef.QName) {
	ar.Called(ctx, app, partID, name)
	ar.mockRunner.newAndRun(ctx, app, partID, ProcessorKind_Actualizer)
}

func (ar *mockActualizerRunner) SetAppPartitions(ap IAppPartitions) {
	ar.Called(ap)
	ar.mockRunner.setAppPartitions(ap)
}

type mockSchedulerRunner struct {
	mock.Mock
	mockRunner
	ISchedulerRunner
}

func (sr *mockSchedulerRunner) NewAndRun(ctx context.Context, app appdef.AppQName, partID istructs.PartitionID, wsIdx istructs.AppWorkspaceNumber, wsid istructs.WSID, job appdef.QName) {
	sr.Called(ctx, app, partID, wsIdx, wsid, job)
	sr.mockRunner.newAndRun(ctx, app, partID, ProcessorKind_Scheduler)
}

func (sr *mockSchedulerRunner) SetAppPartitions(ap IAppPartitions) {
	sr.Called(ap)
	sr.mockRunner.setAppPartitions(ap)
}

func Test_DeployActualizersAndSchedulers(t *testing.T) {
	require := require.New(t)

	appName := istructs.AppQName_test1_app1
	appPartsCount := istructs.NumAppPartitions(10)
	appWSCount := istructs.NumAppWorkspaces(5)

	prj1name := appdef.NewQName("test", "projector1")
	job1name := appdef.NewQName("test", "job1")

	ctx, stop := context.WithCancel(context.Background())

	adb1, appDef1 := func() (appdef.IAppDefBuilder, appdef.IAppDef) {
		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))

		prj := wsb.AddProjector(prj1name)
		prj.SetSync(false)
		prj.Events().Add(appdef.QNameAnyCommand, appdef.ProjectorEventKind_Execute)

		job := wsb.AddJob(job1name)
		job.SetCronSchedule("@every 1s")

		return adb, adb.MustBuild()
	}()

	appConfigs := istructsmem.AppConfigsType{}
	appConfigs.AddBuiltInAppConfig(appName, adb1).SetNumAppWorkspaces(appWSCount)

	appStructs := istructsmem.Provide(
		appConfigs,
		iratesce.TestBucketsFactory,
		payloads.ProvideIAppTokensFactory(itokensjwt.TestTokensJWT()),
		provider.Provide(mem.Provide(), ""))

	mockActualizers := &mockActualizerRunner{}
	mockActualizers.On("SetAppPartitions", mock.Anything).Once()

	mockSchedulers := &mockSchedulerRunner{}
	mockSchedulers.On("SetAppPartitions", mock.Anything).Once()

	appParts, cleanupParts, err := New2(ctx, appStructs, NullSyncActualizerFactory,
		mockActualizers,
		mockSchedulers,
		NullExtensionEngineFactories)
	require.NoError(err)

	defer cleanupParts()

	whatsRun := func() map[istructs.PartitionID]appdef.QNames {
		m := map[istructs.PartitionID]appdef.QNames{}
		for partID := istructs.PartitionID(0); partID < istructs.PartitionID(appPartsCount); partID++ {
			appParts.(*apps).mx.RLock()
			if p, exists := appParts.(*apps).apps[appName].parts[partID]; exists {
				names := p.actualizers.Enum()
				names.Add(appdef.QNamesFromMap(p.schedulers.Enum())...)
				if len(names) > 0 {
					m[partID] = names
				}
			}
			appParts.(*apps).mx.RUnlock()
		}
		return m
	}

	appParts.DeployApp(appName, nil, appDef1, appPartsCount, PoolSize(1, 1, 2, 2), appWSCount)

	t.Run("deploy partitions", func(t *testing.T) {
		parts := make([]istructs.PartitionID, 0, appPartsCount)
		for partID := istructs.PartitionID(0); partID < istructs.PartitionID(appPartsCount); partID++ {
			parts = append(parts, partID)

			mockActualizers.On("NewAndRun", mock.Anything, appName, partID, prj1name).Once()

			ws := schedulers.AppWorkspacesHandledByPartition(appPartsCount, appWSCount, partID)
			for wsID, wsIdx := range ws {
				mockSchedulers.On("NewAndRun", mock.Anything, appName, partID, wsIdx, wsID, job1name).Once()
			}
		}
		appParts.DeployAppPartitions(appName, parts)

		wr := whatsRun()
		require.Len(wr, int(appPartsCount))
		for partID := istructs.PartitionID(0); partID < istructs.PartitionID(appPartsCount); partID++ {
			if len(schedulers.AppWorkspacesHandledByPartition(appPartsCount, appWSCount, partID)) == 0 {
				require.Equal(appdef.QNames{prj1name}, wr[partID])
			} else {
				require.Equal(appdef.QNames{job1name, prj1name}, wr[partID])
			}
		}
	})

	t.Run("redeploy odd partitions", func(t *testing.T) {
		prj2name := appdef.NewQName("test", "projector2")
		job2name := appdef.NewQName("test", "job2")
		appDef2 := func() appdef.IAppDef {
			adb := appdef.New()
			adb.AddPackage("test", "test.com/test")

			wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))

			prj := wsb.AddProjector(prj2name)
			prj.SetSync(false)
			prj.Events().Add(appdef.QNameAnyCommand, appdef.ProjectorEventKind_Execute)

			job := wsb.AddJob(job2name)
			job.SetCronSchedule("@every 1s")

			return adb.MustBuild()
		}()

		t.Run("upgrade app to appDef2", func(t *testing.T) {
			a, ok := appParts.(*apps)
			require.True(ok)

			app1 := a.apps[appName]
			app1.lastestVersion.upgrade(appDef2, app1.lastestVersion.appStructs(), app1.lastestVersion.pools)

			a2, err := appParts.AppDef(appName)
			require.NoError(err)
			require.Equal(appDef2, a2)
		})

		parts := make([]istructs.PartitionID, 0, appPartsCount)
		for partID := istructs.PartitionID(0); partID < istructs.PartitionID(appPartsCount); partID++ {
			if partID%2 == 1 {
				parts = append(parts, partID)

				mockActualizers.On("NewAndRun", mock.Anything, appName, partID, prj2name).Once()

				ws := schedulers.AppWorkspacesHandledByPartition(appPartsCount, appWSCount, partID)
				for wsID, wsIdx := range ws {
					mockSchedulers.On("NewAndRun", mock.Anything, appName, partID, wsIdx, wsID, job2name).Once()
				}
			}
		}
		appParts.DeployAppPartitions(appName, parts)

		wr := whatsRun()
		require.Len(wr, int(appPartsCount))
		for partID := istructs.PartitionID(0); partID < istructs.PartitionID(appPartsCount); partID++ {
			if partID%2 == 1 {
				if len(schedulers.AppWorkspacesHandledByPartition(appPartsCount, appWSCount, partID)) == 0 {
					require.Equal(appdef.QNames{prj2name}, wr[partID])
				} else {
					require.Equal(appdef.QNames{job2name, prj2name}, wr[partID])
				}
			} else {
				if len(schedulers.AppWorkspacesHandledByPartition(appPartsCount, appWSCount, partID)) == 0 {
					require.Equal(appdef.QNames{prj1name}, wr[partID])
				} else {
					require.Equal(appdef.QNames{job1name, prj1name}, wr[partID])
				}
			}
		}
	})

	t.Run("stop vvm from context, wait processors finished, check whatsRun", func(t *testing.T) {
		stop()

		for len(whatsRun()) > 0 {
			time.Sleep(time.Millisecond)
		}
	})
}
