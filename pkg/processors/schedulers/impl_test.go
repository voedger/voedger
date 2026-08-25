/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package schedulers

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"net/url"
	"strings"
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/goutils/testingu"
	"github.com/voedger/voedger/pkg/iextengine"
	iextenginebuiltin "github.com/voedger/voedger/pkg/iextengine/builtin"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/pipeline"
)

func panickingSchedulerExtension(context.Context, iextengine.IExtensionIO) error {
	panic("scheduler extension boom")
}

func TestSchedulerLogging(t *testing.T) {
	appName := istructs.AppQName_test1_app1
	jobQName := appdef.NewQName("test", "TestJob")
	wsid := istructs.WSID(1001)

	adb := builder.New()
	adb.AddPackage("test", "test.com/test")
	adb.AddWorkspace(appdef.NewQName("test", "workspace")).AddJob(jobQName).SetCronSchedule("@every 1m")
	appDef := adb.MustBuild()

	vapp := fmt.Sprintf("vapp=%s", appName)
	wsidStr := fmt.Sprintf("wsid=%d", wsid)
	extension := fmt.Sprintf("extension=job.%s", jobQName)

	t.Run("job.success", func(t *testing.T) {
		logCap := logger.StartCapture(t, logger.LogLevelVerbose)
		mockParts := &mockAppPartitions{appDef: appDef, part: &mockAppPartition{}}

		mockTime := testingu.NewMockTime()
		sr := newSchedulers(BasicSchedulerConfig{Time: mockTime})
		sr.SetAppPartitions(mockParts)

		isolatedTime := sr.SchedulersTime().(testingu.IMockTime)
		isolatedTime.FireNextTimerImmediately()

		vvmCtx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := make(chan struct{})
		go func() {
			defer close(done)
			sr.NewAndRun(vvmCtx, appName, 0, 0, wsid, jobQName)
		}()

		logCap.EventuallyHasLine("job.schedule", vapp, wsidStr, extension)
		logCap.EventuallyHasLine("job.wake-up", vapp, wsidStr, extension)
		logCap.EventuallyHasLine("job.success", vapp, wsidStr, extension)

		cancel()
		<-done
	})

	t.Run("job.error", func(t *testing.T) {
		logCap := logger.StartCapture(t, logger.LogLevelVerbose)
		mockParts := &mockAppPartitions{appDef: appDef, err: errors.New("borrow failed")}

		mockTime := testingu.NewMockTime()
		sr := newSchedulers(BasicSchedulerConfig{Time: mockTime})
		sr.SetAppPartitions(mockParts)

		isolatedTime := sr.SchedulersTime().(testingu.IMockTime)
		isolatedTime.FireNextTimerImmediately()

		vvmCtx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := make(chan struct{})
		go func() {
			defer close(done)
			sr.NewAndRun(vvmCtx, appName, 0, 0, wsid, jobQName)
		}()

		logCap.EventuallyHasLine("job.error", vapp, wsidStr, extension, "borrow failed")

		cancel()
		<-done
	})

	t.Run("built-in extension panic log", func(t *testing.T) {
		logCap := logger.StartCapture(t, logger.LogLevelVerbose)
		fullJobQName := appDef.FullQName(jobQName)
		extEngineFactory := iextenginebuiltin.ProvideExtensionEngineFactory(iextengine.BuiltInAppExtFuncs{
			appName: iextengine.BuiltInExtFuncs{
				fullJobQName: panickingSchedulerExtension,
			},
		}, nil)
		extEngines, err := extEngineFactory.New(context.Background(), appName, nil, nil, 1)
		if err != nil {
			t.Fatal(err)
		}
		extEngine := extEngines[0]
		defer extEngine.Close(context.Background())

		mockParts := &mockAppPartitions{appDef: appDef, part: &mockAppPartition{
			invoke: func(ctx context.Context, _ appdef.QName, state istructs.IState, intents istructs.IIntents) error {
				return extEngine.Invoke(ctx, fullJobQName, iextengine.NewExtensionIO(appDef, state, intents))
			},
		}}

		mockTime := testingu.NewMockTime()
		sr := newSchedulers(BasicSchedulerConfig{Time: mockTime})
		sr.SetAppPartitions(mockParts)

		isolatedTime := sr.SchedulersTime().(testingu.IMockTime)
		isolatedTime.FireNextTimerImmediately()

		vvmCtx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() {
			defer close(done)
			sr.NewAndRun(vvmCtx, appName, 0, 0, wsid, jobQName)
		}()
		defer func() {
			cancel()
			<-done
		}()

		panicMessage := "extension " + fullJobQName.String() + " panic: scheduler extension boom"
		logCap.EventuallyHasLine("level=ERROR", "stage=job.error", vapp, wsidStr, extension, panicMessage, "goroutine ", "[running]:", "panickingSchedulerExtension")

		for _, line := range strings.Split(logCap.String(), "\n") {
			if strings.Contains(line, "stage=job.error") {
				t.Logf("actual built-in extension panic log:\n%s", line)
				break
			}
		}
	})
}

type mockAppPartitions struct {
	appDef appdef.IAppDef
	err    error
	part   appparts.IAppPartition
}

func (m *mockAppPartitions) AppDef(_ appdef.AppQName) (appdef.IAppDef, error) { return m.appDef, nil }
func (m *mockAppPartitions) WaitForBorrow(_ context.Context, _ appdef.AppQName, _ istructs.PartitionID, _ appparts.ProcessorKind) (appparts.IAppPartition, error) {
	return m.part, m.err
}
func (m *mockAppPartitions) DeployApp(_ appdef.AppQName, _ map[string]*url.URL, _ appdef.IAppDef, _ istructs.NumAppPartitions, _ [appparts.ProcessorKind_Count]uint, _ istructs.NumAppWorkspaces) {
	panic("not implemented")
}
func (m *mockAppPartitions) DeployAppPartitions(_ appdef.AppQName, _ []istructs.PartitionID) {
	panic("not implemented")
}
func (m *mockAppPartitions) AppPartsCount(_ appdef.AppQName) (istructs.NumAppPartitions, error) {
	panic("not implemented")
}
func (m *mockAppPartitions) AppWorkspacePartitionID(_ appdef.AppQName, _ istructs.WSID) (istructs.PartitionID, error) {
	panic("not implemented")
}
func (m *mockAppPartitions) Borrow(_ appdef.AppQName, _ istructs.PartitionID, _ appparts.ProcessorKind) (appparts.IAppPartition, error) {
	panic("not implemented")
}
func (m *mockAppPartitions) WorkedActualizers(_ appdef.AppQName) iter.Seq2[istructs.PartitionID, []appdef.QName] {
	return nil
}
func (m *mockAppPartitions) WorkedSchedulers(_ appdef.AppQName) iter.Seq2[istructs.PartitionID, map[appdef.QName][]istructs.WSID] {
	return nil
}
func (m *mockAppPartitions) UpgradeAppDef(_ appdef.AppQName, _ appdef.IAppDef) {
	panic("not implemented")
}

type mockAppPartition struct {
	invoke func(context.Context, appdef.QName, istructs.IState, istructs.IIntents) error
}

func (m *mockAppPartition) App() appdef.AppQName             { panic("not implemented") }
func (m *mockAppPartition) ID() istructs.PartitionID         { panic("not implemented") }
func (m *mockAppPartition) AppStructs() istructs.IAppStructs { return nil }
func (m *mockAppPartition) Release()                         {}
func (m *mockAppPartition) DoSyncActualizer(_ context.Context, _ pipeline.IWorkpiece) error {
	panic("not implemented")
}
func (m *mockAppPartition) Invoke(ctx context.Context, name appdef.QName, state istructs.IState, intents istructs.IIntents) error {
	if m.invoke != nil {
		return m.invoke(ctx, name, state, intents)
	}
	return nil
}
func (m *mockAppPartition) IsOperationAllowed(_ appdef.IWorkspace, _ appdef.OperationKind, _ appdef.QName, _ []appdef.FieldName, _ []appdef.QName) (bool, error) {
	panic("not implemented")
}
func (m *mockAppPartition) IsLimitExceeded(_ appdef.QName, _ appdef.OperationKind, _ istructs.WSID, _ string) (bool, appdef.QName) {
	panic("not implemented")
}
func (m *mockAppPartition) ResetRateLimit(_ appdef.QName, _ appdef.OperationKind, _ istructs.WSID, _ string) {
}
