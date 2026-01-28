/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts_test

import (
	"context"
	"errors"
	"maps"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/appdef/filter"
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/appparts/internal/schedulers"
	"github.com/voedger/voedger/pkg/goutils/testingu"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
	"github.com/voedger/voedger/pkg/iratesce"
	"github.com/voedger/voedger/pkg/isequencer"
	"github.com/voedger/voedger/pkg/istorage/mem"
	"github.com/voedger/voedger/pkg/istorage/provider"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/itokensjwt"
)

type mockRunner struct {
	appParts appparts.IAppPartitions
}

func (mr *mockRunner) newAndRun(ctx context.Context, app appdef.AppQName, partID istructs.PartitionID, kind appparts.ProcessorKind) {
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

func (mr *mockRunner) setAppPartitions(ap appparts.IAppPartitions) {
	mr.appParts = ap
}

type mockActualizerRunner struct {
	mock.Mock
	mockRunner
	appparts.IActualizerRunner
}

func (ar *mockActualizerRunner) NewAndRun(ctx context.Context, app appdef.AppQName, partID istructs.PartitionID, name appdef.QName) {
	ar.Called(ctx, app, partID, name)
	ar.newAndRun(ctx, app, partID, appparts.ProcessorKind_Actualizer)
}

func (ar *mockActualizerRunner) SetAppPartitions(ap appparts.IAppPartitions) {
	ar.Called(ap)
	ar.setAppPartitions(ap)
}

type mockSchedulerRunner struct {
	mock.Mock
	mockRunner
	appparts.ISchedulerRunner
}

func (sr *mockSchedulerRunner) NewAndRun(ctx context.Context, app appdef.AppQName, partID istructs.PartitionID, wsIdx istructs.AppWorkspaceNumber, wsid istructs.WSID, job appdef.QName) {
	sr.Called(ctx, app, partID, wsIdx, wsid, job)
	sr.newAndRun(ctx, app, partID, appparts.ProcessorKind_Scheduler)
}

func (sr *mockSchedulerRunner) SetAppPartitions(ap appparts.IAppPartitions) {
	sr.Called(ap)
	sr.setAppPartitions(ap)
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
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsName := appdef.NewQName("test", "workspace")

		wsb := adb.AddWorkspace(wsName)
		wsb.AddCDoc(appdef.NewQName("test", "WSDesc"))
		wsb.SetDescriptor(appdef.NewQName("test", "WSDesc"))

		_ = wsb.AddCommand(appdef.NewQName("test", "command"))

		prj := wsb.AddProjector(prj1name)
		prj.Events().Add(
			[]appdef.OperationKind{appdef.OperationKind_Execute},
			filter.WSTypes(wsName, appdef.TypeKind_Command))
		prj.SetSync(false)

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
		provider.Provide(mem.Provide(testingu.MockTime), ""), isequencer.SequencesTrustLevel_0, nil)

	mockActualizers := &mockActualizerRunner{}
	mockActualizers.On("SetAppPartitions", mock.Anything).Once()

	mockSchedulers := &mockSchedulerRunner{}
	mockSchedulers.On("SetAppPartitions", mock.Anything).Once()

	appParts, cleanupParts, err := appparts.New2(ctx, appStructs, appparts.NullSyncActualizerFactory,
		mockActualizers,
		mockSchedulers,
		appparts.NullExtensionEngineFactories,
		iratesce.TestBucketsFactory)
	require.NoError(err)

	defer cleanupParts()

	whatsRun := func() map[istructs.PartitionID]appdef.QNames {
		m := map[istructs.PartitionID]appdef.QNames{}
		for pid, actualizers := range appParts.WorkedActualizers(appName) {
			m[pid] = appdef.QNamesFrom(actualizers...)
		}
		for pid, schedulers := range appParts.WorkedSchedulers(appName) {
			names := appdef.QNames{}
			names.Collect(maps.Keys(schedulers))
			if exists, ok := m[pid]; ok {
				names.Add(exists...)
			}
			m[pid] = names
		}
		return m
	}

	appParts.DeployApp(appName, nil, appDef1, appPartsCount, appparts.PoolSize(1, 1, 2, 2), appWSCount)

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
			adb := builder.New()
			adb.AddPackage("test", "test.com/test")

			wsName := appdef.NewQName("test", "workspace")

			wsb := adb.AddWorkspace(wsName)
			wsb.AddCDoc(appdef.NewQName("test", "WSDesc"))
			wsb.SetDescriptor(appdef.NewQName("test", "WSDesc"))

			_ = wsb.AddCommand(appdef.NewQName("test", "command"))

			prj := wsb.AddProjector(prj2name)
			prj.Events().Add(
				[]appdef.OperationKind{appdef.OperationKind_Execute},
				filter.WSTypes(wsName, appdef.TypeKind_Command))
			prj.SetSync(false)

			job := wsb.AddJob(job2name)
			job.SetCronSchedule("@every 1s")

			return adb.MustBuild()
		}()

		t.Run("upgrade app to appDef2", func(t *testing.T) {
			appParts.UpgradeAppDef(appName, appDef2)

			def, err := appParts.AppDef(appName)
			require.NoError(err)
			require.Equal(appDef2, def)
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
