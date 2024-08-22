/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
	"github.com/voedger/voedger/pkg/iratesce"
	"github.com/voedger/voedger/pkg/istorage/mem"
	"github.com/voedger/voedger/pkg/istorage/provider"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/itokensjwt"
)

type mockActualizersRunner struct {
	IActualizerRunner
	mock.Mock
	appParts IAppPartitions
	wg       sync.WaitGroup
}

func (t *mockActualizersRunner) NewAndRun(ctx context.Context, app appdef.AppQName, partID istructs.PartitionID, name appdef.QName) {
	t.wg.Add(1)
	defer t.wg.Done()

	t.Called(ctx, app, partID, name)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// actualizer should be borrowed and released
			p, err := t.appParts.WaitForBorrow(ctx, app, partID, ProcessorKind_Actualizer)
			if err != nil {
				if errors.Is(err, ctx.Err()) {
					return // context canceled while actualizer wait for borrowed partition
				}
				panic(err) // unexpected error while waiting for borrowed partition
			}
			// simulate actualizer work, like p.Invoke(…)
			time.Sleep(time.Millisecond)
			p.Release()
		}
	}
}

func (t *mockActualizersRunner) SetAppPartitions(ap IAppPartitions) {
	t.Called(ap)
	t.appParts = ap
}

func (t *mockActualizersRunner) wait() {
	// the context should be stopped. Here we just wait for actualizers to finish
	t.wg.Wait()
}

func Test_partitionActualizers_deploy(t *testing.T) {
	require := require.New(t)

	prj1name := appdef.NewQName("test", "projector1")

	ctx, stop := context.WithCancel(context.Background())

	adb1, appDef1 := func() (appdef.IAppDefBuilder, appdef.IAppDef) {
		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")

		prj := adb.AddProjector(prj1name)
		prj.SetSync(false)
		prj.Events().Add(appdef.QNameAnyCommand, appdef.ProjectorEventKind_Execute)

		return adb, adb.MustBuild()
	}()

	appConfigs := istructsmem.AppConfigsType{}
	appConfigs.AddBuiltInAppConfig(istructs.AppQName_test1_app1, adb1).SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)

	appStructs := istructsmem.Provide(
		appConfigs,
		iratesce.TestBucketsFactory,
		payloads.TestAppTokensFactory(itokensjwt.TestTokensJWT()),
		provider.Provide(mem.Provide(), ""))

	mockActualizers := &mockActualizersRunner{}
	mockActualizers.On("SetAppPartitions", mock.Anything).Once()

	appParts, cleanupParts, err := New2(ctx, appStructs, NullSyncActualizerFactory,
		mockActualizers,
		NullSchedulerRunner,
		NullExtensionEngineFactories)
	require.NoError(err)

	defer cleanupParts()

	metrics := func() map[istructs.PartitionID]appdef.QNames {
		m := map[istructs.PartitionID]appdef.QNames{}
		for i := istructs.PartitionID(0); i < 10; i++ {
			appParts.(*apps).mx.RLock()
			if p, exists := appParts.(*apps).apps[istructs.AppQName_test1_app1].parts[i]; exists {
				m[i] = p.actualizers.Enum()
			}
			appParts.(*apps).mx.RUnlock()
		}
		return m
	}

	appParts.DeployApp(istructs.AppQName_test1_app1, nil, appDef1, 1, PoolSize(2, 2, 2, 2), istructs.DefaultNumAppWorkspaces)

	t.Run("deploy 10 partitions", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			mockActualizers.On("NewAndRun", mock.Anything, istructs.AppQName_test1_app1, istructs.PartitionID(i), prj1name).Once()
		}
		appParts.DeployAppPartitions(istructs.AppQName_test1_app1, []istructs.PartitionID{0, 1, 2, 3, 4, 5, 6, 7, 8, 9})

		m := metrics()
		require.Len(m, 10)
		for i := istructs.PartitionID(0); i < 10; i++ {
			require.Equal(appdef.QNames{prj1name}, m[i])
		}
	})

	t.Run("redeploy odd partitions", func(t *testing.T) {

		prj2name := appdef.NewQName("test", "projector2")
		appDef2 := func() appdef.IAppDef {
			adb := appdef.New()
			adb.AddPackage("test", "test.com/test")

			prj := adb.AddProjector(prj2name)
			prj.SetSync(false)
			prj.Events().Add(appdef.QNameAnyCommand, appdef.ProjectorEventKind_Execute)

			return adb.MustBuild()
		}()

		t.Run("upgrade test1.app1 to appDef2", func(t *testing.T) {
			a, ok := appParts.(*apps)
			require.True(ok)

			app1 := a.apps[istructs.AppQName_test1_app1]
			app1.lastestVersion.upgrade(appDef2, app1.lastestVersion.appStructs(), app1.lastestVersion.pools)

			a2, err := appParts.AppDef(istructs.AppQName_test1_app1)
			require.NoError(err)
			require.Equal(appDef2, a2)
		})

		for i := 0; i < 10; i++ {
			if i%2 == 1 {
				mockActualizers.On("NewAndRun", mock.Anything, istructs.AppQName_test1_app1, istructs.PartitionID(i), prj2name).Once()
			}
		}
		appParts.DeployAppPartitions(istructs.AppQName_test1_app1, []istructs.PartitionID{1, 3, 5, 7, 9})

		m := metrics()
		require.Len(m, 10)
		for i := istructs.PartitionID(0); i < 10; i++ {
			if i%2 == 1 {
				require.Equal(appdef.QNames{prj2name}, m[i])
			} else {
				require.Equal(appdef.QNames{prj1name}, m[i])
			}
		}
	})

	t.Run("stop vvm from context, wait processors finished, check metrics", func(t *testing.T) {
		stop()

		mockActualizers.wait()

		m := metrics()
		require.Len(m, 10)
		for i := istructs.PartitionID(0); i < 10; i++ {
			require.Empty(m[i])
		}
	})
}
