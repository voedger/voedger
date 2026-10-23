/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 */

package vit

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/coreutils/federation"
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/goutils/timeu"
	"github.com/voedger/voedger/pkg/iblobstorage"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys/collection"
	"github.com/voedger/voedger/pkg/vvm"
	builtinapps "github.com/voedger/voedger/pkg/vvm/builtin"
)

const restartTestInsert = `{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.DocBLOB"}}]}`

func TestVITRestartVVMRejectsBenchmarks(t *testing.T) {
	vit := &VIT{T: new(testing.B)}
	err := vit.RestartVVM(WithAfterStop(func(RestartStorage) {
		t.Fatal("benchmark restart must be rejected before the stopped hook")
	}))
	require.ErrorContains(t, err, "benchmarks")
	require.False(t, vit.serverClosed, "rejected usage must not close the server")
}

func TestVITRestartPreservesStorageByDefault(t *testing.T) {
	for _, ownership := range []struct {
		name   string
		config func(...VITConfigOptFunc) VITConfig
	}{
		{name: "owned", config: NewOwnVITConfig},
		{name: "shared", config: NewSharedVITConfig},
	} {
		t.Run(ownership.name, func(t *testing.T) {
			var factoryCalls, provisioningCalls atomic.Int32
			cfg := ownership.config(
				WithApp(istructs.AppQName_test1_app1, ProvideApp1,
					WithUserLogin("restart-owner", "pwd"),
					WithChildWorkspace(QNameApp1_TestWSKind, "retained-workspace", "", "", "restart-owner", map[string]interface{}{"IntFld": 42})),
				WithVVMConfig(func(cfg *vvm.VVMConfig) {
					factory := cfg.StorageFactory
					cfg.StorageFactory = func(now timeu.ITime) (istorage.IAppStorageFactory, error) {
						factoryCalls.Add(1)
						return factory(now)
					}
				}),
				WithPostInit(func(*VIT) { provisioningCalls.Add(1) }),
			)
			var workspace *AppWorkspace
			var lastInsert *federation.FuncResponse
			vit := runRestartTestPhase(t, &cfg, func(vit *VIT) {
				if cfg.isShared {
					stopSharedVITAfterTest(t, vit)
				}
				workspace = restartTestAppWorkspace(vit)
				configuredWorkspace := vit.WS(istructs.AppQName_test1_app1, "retained-workspace")
				principal := vit.GetPrincipal(istructs.AppQName_test1_app1, "restart-owner")
				suffix := vit.KeyspaceIsolationSuffix
				lastInsert = vit.PostWS(workspace, "c.sys.CUD", restartTestInsert)
				for range 2 {
					previousServer, previousProblem := vit.VoedgerVM, vit.vvmProblemCtx
					previousClient := vit.httpClient
					now, number := vit.Now(), vit.NextNumber()
					principal.Token = "invalidated-before-restart"
					// No preservation flag or stopped hook: retained storage is the default.
					require.NoError(t, vit.RestartVVM())
					require.NotSame(t, previousServer, vit.VoedgerVM)
					require.NotSame(t, previousClient, vit.httpClient)
					requireRestartContextClosed(t, previousProblem)
					require.Equal(t, suffix, vit.KeyspaceIsolationSuffix)
					require.Equal(t, now, vit.Now(), "direct restart must not advance mock time")
					require.Equal(t, number+1, vit.NextNumber())
					require.Same(t, configuredWorkspace, vit.WS(istructs.AppQName_test1_app1, "retained-workspace"))
					require.Same(t, principal, vit.GetPrincipal(istructs.AppQName_test1_app1, "restart-owner"))
					require.NotEqual(t, "invalidated-before-restart", principal.Token)
					vit.PostWS(configuredWorkspace, "q.sys.GRCount", `{"args":{},"elements":[{"fields":["NumGoroutines"]}]}`)
					lastInsert = requireRestartSequenceContinues(t, vit, workspace, lastInsert)
				}
				require.EqualValues(t, 1, factoryCalls.Load(), "restart must reuse the actual initial storage factory")
				require.EqualValues(t, 1, provisioningCalls.Load(), "restart must not reprovision fixture entities")
			})
			if cfg.isShared {
				runRestartTestPhase(t, &cfg, func(reused *VIT) {
					require.Same(t, vit, reused)
					requireRestartRecord(t, reused, workspace, lastInsert.NewID())
					reused.PostWS(workspace, "c.sys.CUD", restartTestInsert)
				})
			} else {
				requireRestartContextClosed(t, vit.vvmProblemCtx)
			}
		})
	}
}

func TestVITRestartWithAnotherConfigPreservesStorage(t *testing.T) {
	var initialAfterTestCalls, replacementAfterTestCalls int
	cfg := newSharedRestartTestConfig(
		WithVVMConfig(func(cfg *vvm.VVMConfig) { cfg.NumQueryProcessors = 3 }),
		WithAfterTest(func(*VIT) { initialAfterTestCalls++ }))
	replacement := NewOwnVITConfig(WithApp(istructs.AppQName_test1_app1, ProvideApp1),
		WithVVMConfig(func(cfg *vvm.VVMConfig) {
			cfg.NumQueryProcessors = 2
			cfg.StorageFactory = func(timeu.ITime) (istorage.IAppStorageFactory, error) {
				panic("replacement configuration must not create another database")
			}
		}),
		WithAfterTest(func(*VIT) { replacementAfterTestCalls++ }))
	initialOptions, replacementOptions := len(cfg.opts), len(replacement.opts)
	var workspace *AppWorkspace
	var id istructs.RecordID
	vit := runRestartTestPhase(t, &cfg, func(vit *VIT) {
		stopSharedVITAfterTest(t, vit)
		workspace = restartTestAppWorkspace(vit)
		id = vit.PostWS(workspace, "c.sys.CUD", restartTestInsert).NewID()
		require.EqualValues(t, 3, vit.NumQueryProcessors)
		previousProblem := vit.vvmProblemCtx
		stopped := false
		require.NoError(t, vit.RestartVVM(WithNewConfig(&replacement), WithAfterStop(func(RestartStorage) {
			stopped = true
			requireRestartContextClosed(t, previousProblem)
		})))
		require.True(t, stopped, "configuration and stopped-hook options must both apply")
		require.EqualValues(t, 2, vit.NumQueryProcessors)
		requireRestartRecord(t, vit, workspace, id)
		require.True(t, cfg.isShared)
		require.False(t, replacement.isShared)
		require.Len(t, cfg.opts, initialOptions)
		require.Len(t, replacement.opts, replacementOptions)
		require.Zero(t, initialAfterTestCalls)
		require.Zero(t, replacementAfterTestCalls)
	})
	require.Equal(t, 1, initialAfterTestCalls, "the active test keeps its original callbacks across restart")
	require.Zero(t, replacementAfterTestCalls)
	runRestartTestPhase(t, &cfg, func(reused *VIT) {
		require.Same(t, vit, reused, "cache ownership remains with the original configuration")
		require.EqualValues(t, 2, reused.NumQueryProcessors)
		requireRestartRecord(t, reused, workspace, id)
	})
	require.Equal(t, 1, initialAfterTestCalls)
	require.Equal(t, 1, replacementAfterTestCalls, "the next acquisition uses the effective configuration's callbacks")
}

func TestVITRestartStoppedHook(t *testing.T) {
	cfg := newSharedRestartTestConfig()
	vit := NewVIT(t, &cfg)
	defer vit.TearDown()
	stopSharedVITAfterTest(t, vit)
	store, err := vit.IAppStorageProvider.AppStorage(istructs.AppQName_test1_app1)
	require.NoError(t, err)
	key := []byte("vit-restart-marker")
	require.NoError(t, store.Put(key, nil, []byte("before restart")))
	previousProblem := vit.vvmProblemCtx
	called := false
	require.NoError(t, vit.RestartVVM(WithAfterStop(func(storage RestartStorage) {
		called = true
		requireRestartContextClosed(t, previousProblem)
		stopped := storage.AppStorage(t, istructs.AppQName_test1_app1)
		requireRestartStorageValue(t, stopped, key, "before restart")
		require.NoError(t, stopped.Put(key, nil, []byte("edited while stopped")))
	})))
	require.True(t, called)
	store, err = vit.IAppStorageProvider.AppStorage(istructs.AppQName_test1_app1)
	require.NoError(t, err)
	requireRestartStorageValue(t, store, key, "edited while stopped")
	vit.PostWS(restartTestAppWorkspace(vit), "c.sys.CUD", restartTestInsert)
}

func TestVITRestartCleanupLifetimes(t *testing.T) {
	var afterTestCalls atomic.Int32
	cfg := newSharedRestartTestConfig(
		WithAfterTest(func(*VIT) { afterTestCalls.Add(1) }))
	reader := &restartReadCloser{Reader: strings.NewReader("")}
	readerAfterRestart := &restartReadCloser{Reader: strings.NewReader("")}
	vit := runRestartTestPhase(t, &cfg, func(vit *VIT) {
		stopSharedVITAfterTest(t, vit)
		workspace := restartTestAppWorkspace(vit)
		offsets := vit.SubscribeForN10n(workspace, collection.QNameCollectionView)
		vit.trackBLOBReader(iblobstorage.BLOBReader{ReadCloser: reader})
		captor := vit.emailCaptor
		require.NoError(t, vit.RestartVVM())
		require.Zero(t, afterTestCalls.Load(), "a direct restart does not end the test phase")
		require.EqualValues(t, 1, reader.closes.Load(), "even an exhausted reader must be closed")
		func() {
			timer := time.NewTimer(testTimeout)
			defer timer.Stop()
			for {
				select {
				case _, open := <-offsets:
					if !open {
						return
					}
				case <-timer.C:
					t.Fatal("old notification subscription remains open after restart")
				}
			}
		}()
		select {
		case _, open := <-captor.emailCaptorCh:
			require.False(t, open, "old email capture is closed after its services stop")
		default:
			t.Fatal("old email capture remains open after restart")
		}
		require.NotSame(t, captor, vit.emailCaptor)
		vit.emailCaptor.Send("", state.EmailMessage{Subject: "restart generation"})
		require.Equal(t, "restart generation", vit.CaptureEmail().Subject)
		vit.trackBLOBReader(iblobstorage.BLOBReader{ReadCloser: readerAfterRestart})
	})
	require.EqualValues(t, 1, afterTestCalls.Load())
	require.EqualValues(t, 1, reader.closes.Load(), "teardown must not close the old reader again")
	require.EqualValues(t, 1, readerAfterRestart.closes.Load(), "teardown must close resources created after restart")
	runRestartTestPhase(t, &cfg, func(reused *VIT) {
		require.Same(t, vit, reused)
	})
	require.EqualValues(t, 2, afterTestCalls.Load())
	require.EqualValues(t, 1, readerAfterRestart.closes.Load(), "a later acquisition must not close an old reader again")
}

func TestVITAfterTestCallbacksReleaseResourcesOnFailure(t *testing.T) {
	for _, failure := range []struct {
		name      string
		fail      func()
		wantPanic any
	}{
		{name: "panic", fail: func() { panic("after-test failure") }, wantPanic: "after-test failure"},
		{name: "Goexit", fail: runtime.Goexit},
	} {
		t.Run(failure.name, func(t *testing.T) {
			reader := &restartReadCloser{Reader: strings.NewReader("")}
			var calls []string
			vit := &VIT{
				T: t,
				currentAfterTest: []func(*VIT){
					func(*VIT) {
						calls = append(calls, "first callback")
						failure.fail()
					},
					func(*VIT) { calls = append(calls, "second callback") },
				},
				currentTestCleanups: []func(){
					func() { calls = append(calls, "unsubscribe") },
					func() { _ = reader.Close() },
					func() { calls = append(calls, "close HTTP client") },
				},
			}
			var panicValue any
			returned := false
			done := make(chan struct{})
			go func() {
				defer close(done)
				defer func() { panicValue = recover() }()
				vit.TearDown()
				returned = true
			}()
			select {
			case <-done:
			case <-time.After(testTimeout):
				t.Fatal("after-test failure blocked resource release")
			}
			require.False(t, returned, "teardown must preserve the callback's panic or Goexit")
			require.Equal(t, failure.wantPanic, panicValue)
			require.Equal(t, []string{"first callback", "second callback", "unsubscribe", "close HTTP client"}, calls)
			require.EqualValues(t, 1, reader.closes.Load())
			vit.TearDown()
			require.Len(t, calls, 4, "repeated teardown must not run callbacks or release resources again")
			require.EqualValues(t, 1, reader.closes.Load())
		})
	}
}

func TestVITRestartResetsStorageDelays(t *testing.T) {
	if coreutils.IsCassandraStorage() {
		t.Skip("storage delays are an in-memory test feature")
	}
	var observed *restartDelayFactory
	cfg := NewOwnVITConfig(WithApp(istructs.AppQName_test1_app1, ProvideApp1),
		WithVVMConfig(func(cfg *vvm.VVMConfig) {
			underlying := cfg.StorageFactory
			cfg.StorageFactory = func(now timeu.ITime) (istorage.IAppStorageFactory, error) {
				factory, err := underlying(now)
				observed = &restartDelayFactory{IAppStorageFactory: factory}
				return observed, err
			}
		}))
	vit := NewVIT(t, &cfg)
	defer vit.TearDown()
	vit.SetMemStorageGetDelay(time.Millisecond)
	vit.SetMemStoragePutDelay(time.Millisecond)
	observed.mu.Lock()
	var delayed []*restartDelayStorage
	for _, storage := range observed.storages {
		if storage.getDelay.Load() != 0 || storage.putDelay.Load() != 0 {
			delayed = append(delayed, storage)
		}
	}
	observed.mu.Unlock()
	require.NotEmpty(t, delayed)
	require.NoError(t, vit.RestartVVM())
	for _, storage := range delayed {
		require.Zero(t, storage.getDelay.Load(), "old generation's Get delay must be reset")
		require.Zero(t, storage.putDelay.Load(), "old generation's Put delay must be reset")
	}
	vit.PostWS(restartTestAppWorkspace(vit), "c.sys.CUD", restartTestInsert)
}

func TestVITRestartFailure(t *testing.T) {
	t.Run("construction error", func(t *testing.T) {
		injected := errors.New("restart storage unavailable")
		factory := &restartFailingStorageFactory{failure: injected}
		vit, requireFailed := startSharedVITForRestartFailure(t,
			WithVVMConfig(func(cfg *vvm.VVMConfig) {
				underlying := cfg.StorageFactory
				cfg.StorageFactory = func(now timeu.ITime) (istorage.IAppStorageFactory, error) {
					var err error
					factory.IAppStorageFactory, err = underlying(now)
					return factory, err
				}
			}))
		err := vit.RestartVVM(WithAfterStop(func(RestartStorage) { factory.fail.Store(true) }))
		require.ErrorIs(t, err, injected)
		requireFailed()
	})

	t.Run("construction panic", func(t *testing.T) {
		vit, requireFailed := startSharedVITForRestartFailure(t)
		replacement := NewOwnVITConfig(WithInit(func() { panic("restart init panic") }))
		require.PanicsWithValue(t, "restart init panic", func() {
			_ = vit.RestartVVM(WithNewConfig(&replacement))
		})
		requireFailed()
	})

	t.Run("launch error", func(t *testing.T) {
		// Reserve the same wildcard address used by the VVM's public listener.
		listener, err := net.Listen("tcp", ":0")
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, listener.Close()) })
		port := listener.Addr().(*net.TCPAddr).Port
		vit, requireFailed := startSharedVITForRestartFailure(t)
		replacement := NewOwnVITConfig(WithApp(istructs.AppQName_test1_app1, ProvideApp1),
			WithVVMConfig(func(cfg *vvm.VVMConfig) { cfg.VVMPort = vvm.VVMPortType(port) }))
		err = vit.RestartVVM(WithNewConfig(&replacement))
		require.Error(t, err, "binding an occupied HTTP port must fail the replacement launch")
		requireFailed()
	})

	t.Run("launch panic", func(t *testing.T) {
		vit, requireFailed := startSharedVITForRestartFailure(t)
		replacement := NewOwnVITConfig(WithApp(istructs.AppQName_test1_app1,
			func(apis builtinapps.APIs, cfg *istructsmem.AppConfigType, ep extensionpoints.IExtensionPoint) builtinapps.Def {
				def := ProvideApp1(apis, cfg, ep)
				def.NumParts++ // Bootstrap rejects a changed persisted partition count.
				return def
			}))
		require.Panics(t, func() { _ = vit.RestartVVM(WithNewConfig(&replacement)) })
		requireFailed()
	})

	t.Run("stopped callback panic", func(t *testing.T) {
		vit, requireFailed := startSharedVITForRestartFailure(t)
		require.PanicsWithValue(t, "stopped callback panic", func() {
			_ = vit.RestartVVM(WithAfterStop(func(RestartStorage) { panic("stopped callback panic") }))
		})
		requireFailed()
	})
}

func TestVITRestartCallbackFailNow(t *testing.T) {
	const childEnv = "VOEDGER_VIT_RESTART_FAILNOW_CHILD"
	const cleanupMarker = "restart FailNow cleanup verified"
	if os.Getenv(childEnv) == "1" {
		vit, requireFailed := startSharedVITForRestartFailure(t)
		t.Cleanup(func() {
			requireFailed()
			fmt.Println(cleanupMarker)
		})
		_ = vit.RestartVVM(WithAfterStop(func(RestartStorage) { t.FailNow() }))
		t.Fatal("FailNow returned to the test body")
	}
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestVITRestartCallbackFailNow$", "-test.v")
	command.Env = append(os.Environ(), childEnv+"=1")
	output, err := command.CombinedOutput()
	require.NoError(t, ctx.Err(), "%s", output)
	require.Error(t, err, "the child test deliberately calls FailNow")
	require.Contains(t, string(output), cleanupMarker)
	require.NotContains(t, string(output), "FailNow returned to the test body")
}

func TestVITSharedRestartAdapters(t *testing.T) {
	for _, adapter := range []string{"callbacks", "stopped hook", "final stopped hook"} {
		t.Run(adapter, func(t *testing.T) {
			var launches, afterTestCalls atomic.Int32
			cfg := newSharedRestartTestConfig(
				WithInit(func() { launches.Add(1) }),
				WithAfterTest(func(*VIT) { afterTestCalls.Add(1) }))
			var first *VIT
			var workspace *AppWorkspace
			var before *federation.FuncResponse
			var firstTime time.Time
			var secondProblem context.Context
			var hooks []string
			firstRun := func(t *testing.T, vit *VIT) {
				first = vit
				stopSharedVITAfterTest(t, first)
				workspace = restartTestAppWorkspace(vit)
				before = vit.PostWS(workspace, "c.sys.CUD", restartTestInsert)
				firstTime = vit.Now()
				require.Zero(t, afterTestCalls.Load())
				hooks = append(hooks, "first")
			}
			secondRun := func(t *testing.T, vit *VIT) {
				require.Same(t, first, vit)
				require.Equal(t, firstTime.Add(day), vit.Now())
				require.EqualValues(t, 1, afterTestCalls.Load())
				requireRestartSequenceContinues(t, vit, workspace, before)
				secondProblem = vit.vvmProblemCtx
				hooks = append(hooks, "second")
			}
			if adapter == "callbacks" {
				TestRestartPreservingStorage(t, &cfg, firstRun, secondRun)
				require.Equal(t, []string{"first", "second"}, hooks)
			} else {
				callbacks := RestartPreservingStorageHooks{
					FirstRun: firstRun,
					AfterFirstStop: func(t *testing.T, storage RestartStorage) {
						requireRestartContextClosed(t, first.vvmProblemCtx)
						require.NotNil(t, storage.AppStorage(t, istructs.AppQName_test1_app1))
						hooks = append(hooks, "first stopped")
					},
					SecondRun: secondRun,
				}
				if adapter == "final stopped hook" {
					callbacks.AfterSecondStop = func(t *testing.T, storage RestartStorage) {
						requireRestartContextClosed(t, secondProblem)
						require.NotNil(t, storage.AppStorage(t, istructs.AppQName_test1_app1))
						hooks = append(hooks, "second stopped")
					}
				}
				TestRestartPreservingStorageWithHooks(t, &cfg, callbacks)
				expected := []string{"first", "first stopped", "second"}
				if adapter == "final stopped hook" {
					expected = append(expected, "second stopped")
				}
				require.Equal(t, expected, hooks)
			}
			require.EqualValues(t, 2, afterTestCalls.Load())
			expectedLaunches := int32(2)
			if adapter == "final stopped hook" {
				expectedLaunches++
			}
			require.Equal(t, expectedLaunches, launches.Load())
			runRestartTestPhase(t, &cfg, func(reused *VIT) {
				require.Same(t, first, reused)
				require.Equal(t, expectedLaunches, launches.Load(), "reacquisition must not start another VVM")
				requireRestartRecord(t, reused, workspace, before.NewID())
			})
		})
	}
}

func newSharedRestartTestConfig(opts ...VITConfigOptFunc) VITConfig {
	return NewSharedVITConfig(append([]VITConfigOptFunc{
		WithApp(istructs.AppQName_test1_app1, ProvideApp1),
	}, opts...)...)
}

func runRestartTestPhase(t *testing.T, cfg *VITConfig, run func(*VIT)) *VIT {
	t.Helper()
	vit := NewVIT(t, cfg)
	defer vit.TearDown()
	run(vit)
	return vit
}

func stopSharedVITAfterTest(t *testing.T, vit *VIT) {
	t.Helper()
	// These configurations are local to their tests. Stop the current runtime
	// after all phase releases and reacquisition checks have completed.
	t.Cleanup(func() { require.NoError(t, vit.Shutdown()) })
}

func restartTestAppWorkspace(vit *VIT) *AppWorkspace {
	return &AppWorkspace{
		WorkspaceDescriptor: WorkspaceDescriptor{
			WSID: istructs.NewWSID(istructs.CurrentClusterID(), istructs.FirstBaseAppWSID),
		},
		Owner: vit.GetSystemPrincipal(istructs.AppQName_test1_app1),
	}
}

func requireRestartRecord(t *testing.T, vit *VIT, workspace *AppWorkspace, id istructs.RecordID) {
	t.Helper()
	record := vit.SQLQuery(workspace, "select * from app1pkg.DocBLOB where id = %d", id)
	require.EqualValues(t, id, record["sys.ID"])
	require.Equal(t, "app1pkg.DocBLOB", record["sys.QName"])
}

func requireRestartSequenceContinues(t *testing.T, vit *VIT, workspace *AppWorkspace, before *federation.FuncResponse) *federation.FuncResponse {
	t.Helper()
	requireRestartRecord(t, vit, workspace, before.NewID())
	after := vit.PostWS(workspace, "c.sys.CUD", restartTestInsert)
	require.Equal(t, before.CurrentWLogOffset+1, after.CurrentWLogOffset)
	require.Equal(t, before.NewID()+1, after.NewID())
	return after
}

func requireRestartStorageValue(t *testing.T, storage istorage.IAppStorage, key []byte, expected string) {
	t.Helper()
	var value []byte
	found, err := storage.Get(key, nil, &value)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, expected, string(value))
}

func requireRestartContextClosed(t *testing.T, ctx context.Context) {
	t.Helper()
	select {
	case <-ctx.Done():
	default:
		t.Fatal("previous VVM context remains open")
	}
}

// startSharedVITForRestartFailure returns the running VIT and an assertion for
// its terminal state. The caller can run that assertion from a FailNow cleanup.
func startSharedVITForRestartFailure(t *testing.T, opts ...VITConfigOptFunc) (*VIT, func()) {
	t.Helper()
	cfg := newSharedRestartTestConfig(opts...)
	vit := NewVIT(t, &cfg)
	previousProblem := vit.vvmProblemCtx
	return vit, func() {
		t.Helper()
		requireRestartContextClosed(t, previousProblem)
		require.Error(t, vit.RestartVVM(), "failed VIT must reject another restart")
		require.NotPanics(t, vit.TearDown)
		// Match NewVIT's testing.TB-based error reporting without making an expected
		// failure abort the parent test. Existing VIT tests also use a separate testing.T.
		rejectedTB := new(testing.T)
		done := make(chan struct{})
		var acquired *VIT
		var panicValue interface{}
		go func() {
			defer close(done)
			defer func() { panicValue = recover() }()
			acquired = NewVIT(rejectedTB, &cfg)
		}()
		select {
		case <-done:
		case <-time.After(testTimeout):
			t.Fatal("failed shared identity blocked reacquisition instead of rejecting it")
		}
		if acquired != nil {
			acquired.T = t
			acquired.TearDown()
			require.NoError(t, acquired.Shutdown())
			t.Fatal("failed shared identity was silently recreated")
		}
		require.True(t, rejectedTB.Failed() || panicValue != nil, "reacquisition must reject the closed server")
	}
}

type restartReadCloser struct {
	io.Reader
	closes atomic.Int32
}

func (r *restartReadCloser) Close() error {
	r.closes.Add(1)
	return nil
}

type restartFailingStorageFactory struct {
	istorage.IAppStorageFactory
	fail    atomic.Bool
	failure error
}

func (f *restartFailingStorageFactory) AppStorage(name istorage.SafeAppName) (istorage.IAppStorage, error) {
	if f.fail.Load() {
		return nil, f.failure
	}
	return f.IAppStorageFactory.AppStorage(name)
}

type restartDelayFactory struct {
	istorage.IAppStorageFactory
	mu       sync.Mutex
	storages []*restartDelayStorage
}

func (f *restartDelayFactory) AppStorage(name istorage.SafeAppName) (istorage.IAppStorage, error) {
	storage, err := f.IAppStorageFactory.AppStorage(name)
	if err != nil {
		return nil, err
	}
	setter, ok := storage.(istorage.IStorageDelaySetter)
	if !ok {
		return nil, fmt.Errorf("storage %s does not support test delays", name)
	}
	observed := &restartDelayStorage{IAppStorage: storage, setter: setter}
	f.mu.Lock()
	f.storages = append(f.storages, observed)
	f.mu.Unlock()
	return observed, nil
}

type restartDelayStorage struct {
	istorage.IAppStorage
	setter   istorage.IStorageDelaySetter
	getDelay atomic.Int64
	putDelay atomic.Int64
}

func (s *restartDelayStorage) SetTestDelayGet(delay time.Duration) {
	s.getDelay.Store(int64(delay))
	s.setter.SetTestDelayGet(delay)
}

func (s *restartDelayStorage) SetTestDelayPut(delay time.Duration) {
	s.putDelay.Store(int64(delay))
	s.setter.SetTestDelayPut(delay)
}
