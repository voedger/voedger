/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts_test

import (
	"context"
	"errors"
	"maps"
	"net/url"
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
	"github.com/voedger/voedger/pkg/goutils/timeu"
	"github.com/voedger/voedger/pkg/iextengine"
	iextenginebuiltin "github.com/voedger/voedger/pkg/iextengine/builtin"
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

func (sr *mockSchedulerRunner) SchedulersTime() timeu.ITime {
	return testingu.MockTime
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

type errorExtensionEngineFactory struct{ err error }

func (f errorExtensionEngineFactory) New(context.Context, appdef.AppQName, []iextengine.ExtensionModule, *iextengine.ExtEngineConfig, uint) ([]iextengine.IExtensionEngine, error) {
	return nil, f.err
}

func TestDeployApp_ValidateExtensions_MatchVSQLAndCode(t *testing.T) {
	const pkgLocal = "test"
	const pkgPath = "test.com/test"
	appName := istructs.AppQName_test1_app1

	cmdName := appdef.NewQName(pkgLocal, "cmd")
	qryName := appdef.NewQName(pkgLocal, "qry")
	prjName := appdef.NewQName(pkgLocal, "prj")
	jobName := appdef.NewQName(pkgLocal, "job")

	cmdFQN := appdef.NewFullQName(pkgPath, "cmd")
	qryFQN := appdef.NewFullQName(pkgPath, "qry")
	prjFQN := appdef.NewFullQName(pkgPath, "prj")
	jobFQN := appdef.NewFullQName(pkgPath, "job")

	buildAppDef := func() appdef.IAppDef {
		adb := builder.New()
		adb.AddPackage(pkgLocal, pkgPath)
		wsName := appdef.NewQName(pkgLocal, "workspace")
		wsb := adb.AddWorkspace(wsName)
		wsb.AddCDoc(appdef.NewQName(pkgLocal, "WSDesc"))
		wsb.SetDescriptor(appdef.NewQName(pkgLocal, "WSDesc"))
		wsb.AddCommand(cmdName)
		wsb.AddQuery(qryName)
		prj := wsb.AddProjector(prjName)
		prj.Events().Add([]appdef.OperationKind{appdef.OperationKind_Execute},
			filter.WSTypes(wsName, appdef.TypeKind_Command))
		wsb.AddJob(jobName).SetCronSchedule("@every 1s")
		return adb.MustBuild()
	}

	noopExt := func(context.Context, iextengine.IExtensionIO) error { return nil }

	deploy := func(t *testing.T, def appdef.IAppDef, eef iextengine.ExtensionEngineFactories, extModuleURLs map[string]*url.URL) (panicErr error) {
		t.Helper()
		appConfigs := istructsmem.AppConfigsType{}
		appConfigs.AddBuiltInAppConfig(appName, builder.New()).SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
		appStructs := istructsmem.Provide(
			appConfigs,
			payloads.ProvideIAppTokensFactory(itokensjwt.TestTokensJWT()),
			provider.Provide(mem.Provide(testingu.MockTime), ""), isequencer.SequencesTrustLevel_0, nil)
		ctx, cancel := context.WithCancel(context.Background())
		appParts, cleanup, err := appparts.New2(ctx, appStructs,
			appparts.NullSyncActualizerFactory, appparts.NullActualizerRunner, appparts.NullSchedulerRunner,
			eef, iratesce.TestBucketsFactory)
		require.New(t).NoError(err)
		defer func() {
			cancel()
			cleanup()
		}()
		defer func() {
			if r := recover(); r != nil {
				panicErr, _ = r.(error)
			}
		}()
		appParts.DeployApp(appName, extModuleURLs, def, 1, appparts.PoolSize(1, 1, 1, 1), istructs.DefaultNumAppWorkspaces)
		return nil
	}

	makeEEF := func(funcs iextengine.BuiltInAppExtFuncs, stateless iextengine.BuiltInExtFuncs) iextengine.ExtensionEngineFactories {
		return iextengine.ExtensionEngineFactories{
			appdef.ExtensionEngineKind_BuiltIn: iextenginebuiltin.ProvideExtensionEngineFactory(funcs, stateless),
			appdef.ExtensionEngineKind_WASM:    iextengine.NullExtensionEngineFactory,
		}
	}

	t.Run("aligned set: per-app builtin funcs match vsql", func(t *testing.T) {
		require := require.New(t)
		err := deploy(t, buildAppDef(), makeEEF(
			iextengine.BuiltInAppExtFuncs{appName: {cmdFQN: noopExt, qryFQN: noopExt, prjFQN: noopExt, jobFQN: noopExt}},
			nil), nil)
		require.NoError(err)
	})

	t.Run("aligned set: stateless funcs match vsql", func(t *testing.T) {
		require := require.New(t)
		err := deploy(t, buildAppDef(), makeEEF(
			nil,
			iextengine.BuiltInExtFuncs{cmdFQN: noopExt, qryFQN: noopExt, prjFQN: noopExt, jobFQN: noopExt}), nil)
		require.NoError(err)
	})

	t.Run("missing per-app builtin code", func(t *testing.T) {
		require := require.New(t)
		err := deploy(t, buildAppDef(), makeEEF(
			iextengine.BuiltInAppExtFuncs{appName: {cmdFQN: noopExt, qryFQN: noopExt, prjFQN: noopExt}},
			nil), nil)
		require.ErrorIs(err, appparts.ErrDeployment)
		require.ErrorContains(err, "in vsql, not in code")
		require.ErrorContains(err, jobName.String())
	})

	t.Run("missing stateless code", func(t *testing.T) {
		require := require.New(t)
		err := deploy(t, buildAppDef(), makeEEF(
			nil,
			iextengine.BuiltInExtFuncs{cmdFQN: noopExt, qryFQN: noopExt, prjFQN: noopExt}), nil)
		require.ErrorIs(err, appparts.ErrDeployment)
		require.ErrorContains(err, jobName.String())
	})

	t.Run("extra per-app code not in vsql", func(t *testing.T) {
		require := require.New(t)
		extraFQN := appdef.NewFullQName(pkgPath, "extra")
		err := deploy(t, buildAppDef(), makeEEF(
			iextengine.BuiltInAppExtFuncs{appName: {cmdFQN: noopExt, qryFQN: noopExt, prjFQN: noopExt, jobFQN: noopExt, extraFQN: noopExt}},
			nil), nil)
		require.ErrorIs(err, appparts.ErrDeployment)
		require.ErrorContains(err, "in code, not in vsql")
		require.ErrorContains(err, extraFQN.String())
	})

	t.Run("extra stateless in known package fails", func(t *testing.T) {
		require := require.New(t)
		extraFQN := appdef.NewFullQName(pkgPath, "extra")
		err := deploy(t, buildAppDef(), makeEEF(
			iextengine.BuiltInAppExtFuncs{appName: {cmdFQN: noopExt, qryFQN: noopExt, prjFQN: noopExt, jobFQN: noopExt}},
			iextengine.BuiltInExtFuncs{extraFQN: noopExt}), nil)
		require.ErrorIs(err, appparts.ErrDeployment)
		require.ErrorContains(err, "in code, not in vsql")
	})

	t.Run("extra stateless in unknown package is allowed", func(t *testing.T) {
		require := require.New(t)
		err := deploy(t, buildAppDef(), makeEEF(
			iextengine.BuiltInAppExtFuncs{appName: {cmdFQN: noopExt, qryFQN: noopExt, prjFQN: noopExt, jobFQN: noopExt}},
			iextengine.BuiltInExtFuncs{appdef.NewFullQName("other.com/pkg", "ext"): noopExt}), nil)
		require.NoError(err)
	})

	t.Run("WASM-engine factory error wrapped as ErrDeployment", func(t *testing.T) {
		require := require.New(t)
		const wasmPkg = "wasmpkg"
		const wasmPath = "test.com/wasm"
		wasmCmdName := appdef.NewQName(wasmPkg, "wasmcmd")
		adb := builder.New()
		adb.AddPackage(wasmPkg, wasmPath)
		wsName := appdef.NewQName(wasmPkg, "ws")
		wsb := adb.AddWorkspace(wsName)
		wsb.AddCDoc(appdef.NewQName(wasmPkg, "WSDesc"))
		wsb.SetDescriptor(appdef.NewQName(wasmPkg, "WSDesc"))
		wsb.AddCommand(wasmCmdName).SetEngine(appdef.ExtensionEngineKind_WASM)
		def := adb.MustBuild()
		moduleURL, _ := url.Parse("file:///fake.wasm")
		eef := iextengine.ExtensionEngineFactories{
			appdef.ExtensionEngineKind_BuiltIn: iextenginebuiltin.ProvideExtensionEngineFactory(nil, nil),
			appdef.ExtensionEngineKind_WASM:    errorExtensionEngineFactory{err: errors.New("wasm boom")},
		}
		err := deploy(t, def, eef, map[string]*url.URL{wasmPath: moduleURL})
		require.ErrorIs(err, appparts.ErrDeployment)
		require.ErrorContains(err, "wasm boom")
	})
}
