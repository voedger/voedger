/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */
package elections

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istorage/bbolt"
	"github.com/voedger/voedger/pkg/istorage/cas"
	"github.com/voedger/voedger/pkg/istorage/mem"
	"github.com/voedger/voedger/pkg/istorage/provider"
	"github.com/voedger/voedger/pkg/vvm/storage"
)

// this used as pKey. Actual value does not matter in tests
var pKeyPrefix = []byte{1}

const seconds10 = 10

func TestElections_BasicUsage(t *testing.T) {
	restore := logger.SetLogLevelWithRestore(logger.LogLevelVerbose)
	defer restore()

	iTTLStorage, appStorage := newMockStorage()
	elector, cleanup := Provide(iTTLStorage, coreutils.MockTime)
	defer cleanup()

	ctx := elector.AcquireLeadership("testKey", "leaderVal", seconds10)
	require.NotNil(t, ctx, "Should return a non-nil context on successful acquisition")

	storedData := []byte{}
	ok, err := appStorage.TTLGet(pKeyPrefix, []byte("testKey"), &storedData)
	require.NoError(t, err)
	require.True(t, ok, "Key should be in store")
	require.Equal(t, "leaderVal", string(storedData))

	// Release leadership
	elector.ReleaseLeadership("testKey")
	<-ctx.Done()
}

func TestElections_AcquireLeadership_Failure(t *testing.T) {
	restore := logger.SetLogLevelWithRestore(logger.LogLevelVerbose)
	defer restore()

	iTTLStorage, appStorage := newMockStorage()

	elector, cleanup := Provide(iTTLStorage, coreutils.MockTime)
	defer cleanup()

	require.NoError(t, appStorage.Put(pKeyPrefix, []byte("occupied"), []byte("someVal")))

	ctx := elector.AcquireLeadership("occupied", "myVal", seconds10)
	require.Nil(t, ctx, "Should return nil if the key is already in storage")
}

func TestElections_AcquireLeadership_Error(t *testing.T) {
	restore := logger.SetLogLevelWithRestore(logger.LogLevelVerbose)
	defer restore()

	iTTLStorage, _ := newMockStorage()

	iTTLStorage.(*mockStorage).errorTrigger["badKey"] = true

	elector, cleanup := Provide(iTTLStorage, coreutils.MockTime)
	defer cleanup()

	ctx := elector.AcquireLeadership("badKey", "val", seconds10)
	require.Nil(t, ctx, "Should return nil if a storage error occurs")
}

func TestElections_CompareAndSwap_RenewFails(t *testing.T) {
	restore := logger.SetLogLevelWithRestore(logger.LogLevelVerbose)
	defer restore()

	iTTLStorage, appStorage := newMockStorage()

	elector, cleanup := Provide(iTTLStorage, coreutils.MockTime)
	defer cleanup()

	ctx := elector.AcquireLeadership("renewKey", "renewVal", seconds10)
	require.NotNil(t, ctx)

	// sabotage the storage so next CompareAndSwap fails by changing the value
	require.NoError(t, appStorage.Put(pKeyPrefix, []byte("renewKey"), []byte("someOtherVal")))

	// trigger the renewal
	coreutils.MockTime.Sleep((seconds10 + 1) * time.Second)

	// The leadership is forcibly released in the background.
	<-ctx.Done()
	require.ErrorIs(t, ctx.Err(), context.Canceled, "Context should be canceled after renewal failure")

	keptValue := []byte{}
	ok, err := appStorage.Get(pKeyPrefix, []byte("renewKey"), &keptValue)
	require.NoError(t, err)
	require.True(t, ok, "Key should remain but with different value => we lost leadership.")
	require.Equal(t, "someOtherVal", string(keptValue))
}

func TestElections_ReleaseLeadership_NoLeader(t *testing.T) {
	restore := logger.SetLogLevelWithRestore(logger.LogLevelVerbose)
	defer restore()

	iTTLStorage, _ := newMockStorage()

	elector, cleanup := Provide(iTTLStorage, coreutils.MockTime)
	defer cleanup()

	// Releasing unknown key => no effect
	elector.ReleaseLeadership("unknownKey")
}

func TestElections_CleanupDisallowsNew(t *testing.T) {
	restore := logger.SetLogLevelWithRestore(logger.LogLevelVerbose)
	defer restore()

	iTTLStorage, _ := newMockStorage()

	elector, closeFunc := Provide(iTTLStorage, coreutils.MockTime)

	ctx1 := elector.AcquireLeadership("keyA", "valA", seconds10)
	require.NotNil(t, ctx1)

	closeFunc() // cleanup => no further acquisitions

	<-ctx1.Done()
	ctx2 := elector.AcquireLeadership("keyB", "valB", seconds10)
	require.Nil(t, ctx2, "No new leadership after cleanup")
}

// TestElections_LeadershipExpires ensures that if time passes beyond the TTL and we do not renew,
// the key is automatically pruned from storage, making renewal fail.
func TestElections_LeadershipExpires(t *testing.T) {
	restore := logger.SetLogLevelWithRestore(logger.LogLevelVerbose)
	defer restore()

	iTTLStorage, _ := newMockStorage()
	elector, cleanup := Provide(iTTLStorage, coreutils.MockTime)
	defer cleanup()
	iTTLStorage.(*mockStorage).onBeforeCompareAndSwap = func() {
		if iTTLStorage.(*mockStorage).requiresRealTime {
			time.Sleep(seconds10 * time.Second)
		}
	}

	// Acquire leadership with a short TTL
	ctx := elector.AcquireLeadership("expireKey", "expireVal", seconds10)
	require.NotNil(t, ctx, "Should have leadership initially")

	coreutils.MockTime.Sleep(seconds10 * time.Second)

	<-ctx.Done()
}

func TestCleanupDuringRenewal(t *testing.T) {
	restore := logger.SetLogLevelWithRestore(logger.LogLevelVerbose)
	defer restore()

	iTTLStorage, _ := newMockStorage()
	compareAndSwapCalled := make(chan interface{})
	elector, cleanup := Provide(iTTLStorage, coreutils.MockTime)
	defer cleanup()

	iTTLStorage.(*mockStorage).onBeforeCompareAndSwap = func() {
		close(compareAndSwapCalled)
	}

	ctx := elector.AcquireLeadership("expireKey", "expireVal", seconds10)

	{
		coreutils.MockTime.Sleep(time.Duration(seconds10) * time.Second)

		// gauarantee that <-ticker case is fired, not <-li.ctx.Done()
		// otherwise we do not know which case will fire on cleanup()
		<-compareAndSwapCalled

		// now force cancel everything.
		// successful finalizing after that shows that there are no deadlocks caused by simultaneous locks in
		// cleanup() and in releaseLeadership() after leadership loss
		cleanup()
		<-ctx.Done()
	}
}

// mockStorage is a thread-safe in-memory mock of ITTLStorage that supports key expiration.
type mockStorage struct {
	pKeyPrefix             []byte
	storage                storage.IVVMAppTTLStorage
	errorTrigger           map[string]bool
	onBeforeCompareAndSwap func() // != nil -> called right before CompareAndSwap. Need to implement hook in tests
	requiresRealTime       bool   // true -> real time should be used on awaiting, otherwise MockTime.Sleep()
}

// causeErrorIfNeeded simulates a forced error for specific keys
func (m *mockStorage) causeErrorIfNeeded(key string) error {
	if m.errorTrigger[key] {
		return errors.New("forced storage error for test")
	}
	return nil
}

// InsertIfNotExist checks for expiration, then inserts key if absent, setting its TTL.
func (m *mockStorage) InsertIfNotExist(key string, val string, ttlSeconds int) (bool, error) {
	if err := m.causeErrorIfNeeded(key); err != nil {
		return false, err
	}

	return m.storage.InsertIfNotExists(m.pKeyPrefix, []byte(key), []byte(val), ttlSeconds)
}

// CompareAndSwap refreshes TTL if oldVal matches the current value. Otherwise fails.
func (m *mockStorage) CompareAndSwap(key string, oldVal string, newVal string, ttlSeconds int) (bool, error) {
	if m.onBeforeCompareAndSwap != nil {
		m.onBeforeCompareAndSwap()
	}

	if err := m.causeErrorIfNeeded(key); err != nil {
		return false, err
	}

	return m.storage.CompareAndSwap(m.pKeyPrefix, []byte(key), []byte(oldVal), []byte(newVal), ttlSeconds)
}

// CompareAndDelete removes the key if current value matches val. Expiration is also removed.
func (m *mockStorage) CompareAndDelete(key string, val string) (bool, error) {
	if err := m.causeErrorIfNeeded(key); err != nil {
		return false, err
	}

	return m.storage.CompareAndDelete(m.pKeyPrefix, []byte(key), []byte(val))
}

func newMockStorage() (storage.ITTLStorage[string, string], istorage.IAppStorage) {
	return newMockStorage_mem()
	// return newMockStorage_cas()
	// return newMockStorage_bbolt()
}

func newMockStorage_bbolt() (storage.ITTLStorage[string, string], istorage.IAppStorage) {
	asp := provider.Provide(bbolt.Provide(bbolt.ParamsType{
		DBDir: "c:/workspace/bbolttest",
	}, coreutils.MockTime))
	appStorage, err := asp.AppStorage(appdef.NewAppQName(appdef.SysOwner, "vvm"))
	if err != nil {
		panic(err)
	}
	return &mockStorage{
		pKeyPrefix:   pKeyPrefix,
		storage:      appStorage,
		errorTrigger: map[string]bool{},
	}, appStorage
}

func newMockStorage_mem() (storage.ITTLStorage[string, string], istorage.IAppStorage) {
	asp := provider.Provide(mem.Provide(coreutils.MockTime))
	appStorage, err := asp.AppStorage(appdef.NewAppQName(appdef.SysOwner, "vvm"))
	if err != nil {
		panic(err)
	}
	return &mockStorage{
		pKeyPrefix:   pKeyPrefix,
		storage:      appStorage,
		errorTrigger: map[string]bool{},
	}, appStorage
}

func newMockStorage_cas() (storage.ITTLStorage[string, string], istorage.IAppStorage) {
	appStorageFactory, err := cas.Provide(cas.CassandraParamsType{
		Hosts:                   "127.0.0.1",
		Port:                    9042,
		KeyspaceWithReplication: cas.SimpleWithReplication,
	})
	asp := provider.Provide(appStorageFactory)
	appStorage, err := asp.AppStorage(appdef.NewAppQName(appdef.SysOwner, "vvm"))
	if err != nil {
		panic(err)
	}
	return &mockStorage{
		pKeyPrefix:       pKeyPrefix,
		storage:          appStorage,
		errorTrigger:     map[string]bool{},
		requiresRealTime: true,
	}, appStorage
}
