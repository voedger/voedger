/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */
package vvm

import (
	"encoding/binary"
	"log"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/coreutils/utils"
	"github.com/voedger/voedger/pkg/goutils/testingu"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
)

func TestBasic(t *testing.T) {
	t.Run("VVMStartAndStop", func(t *testing.T) {
		r := require.New(t)

		vvmCfg1 := GetTestVVMCfg(net.IPv4(192, 168, 0, 1))
		vvm1, err := Provide(vvmCfg1)
		r.NoError(err)

		// Launch VVM1
		problemCtx := vvm1.Launch(DefaultLeadershipDurationSeconds, DefaultLeadershipAcquisitionDuration)
		r.NoError(problemCtx.Err())
		r.NoError(vvm1.Shutdown())
		<-vvm1.shutdownedCtx.Done()
	})

	// [~server.design.orch/VVM.test.Basic~impl]
	t.Run("LeadershipCollision", func(t *testing.T) {
		r := require.New(t)

		iTime := testingu.MockTime
		vvmCfg1 := GetTestVVMCfg(net.IPv4(192, 168, 0, 1))

		// make so that VVM launch on vvmCfg1 will store the resulting storage in sharedStorageFactory
		suffix := t.Name() + uuid.NewString()
		sharedStorageFactory, err := vvmCfg1.StorageFactory()
		require.NoError(t, err)
		vvmCfg1.KeyspaceNameSuffix = suffix
		vvmCfg1.StorageFactory = func() (istorage.IAppStorageFactory, error) {
			return sharedStorageFactory, nil
		}

		vvm1, err := Provide(vvmCfg1)
		r.NoError(err)

		// Launch VVM1
		problemCtx1 := vvm1.Launch(DefaultLeadershipDurationSeconds, DefaultLeadershipAcquisitionDuration)
		r.NoError(problemCtx1.Err())

		// Launch VVM2, expecting leadership acquisition to fail
		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()

			vvmCfg2 := GetTestVVMCfg(net.IPv4(192, 168, 0, 2))

			// set vvmCfg2 storage factory to the one from vvm1
			vvmCfg2.StorageFactory = func() (provider istorage.IAppStorageFactory, err error) {
				return sharedStorageFactory, nil
			}
			vvmCfg2.KeyspaceNameSuffix = suffix

			vvm2, err := Provide(vvmCfg2)
			r.NoError(err)

			go func() {
				// force case <-leadershipAcquistionTimerCh to fire in tryToAcquireLeadership()
				<-vvm2.leadershipAcquisitionTimerArmed
				iTime.Sleep(2 * time.Second)
			}()

			problemCtx2 := vvm2.Launch(DefaultLeadershipDurationSeconds, LeadershipAcquisitionDuration(time.Second))

			<-problemCtx2.Done()

			r.Error(problemCtx2.Err(), "VVM2 should not acquire leadership")
			r.ErrorIs(vvm2.Shutdown(), ErrVVMLeadershipAcquisition)
		}()

		wg.Wait()
		r.NoError(vvm1.Shutdown())
	})
}

// [~server.design.orch/VVM.test.CancelLeadership~impl]
func TestCancelLeadershipOnManualShutdown(t *testing.T) {
	r := require.New(t)

	vvmCfg := GetTestVVMCfg(net.IPv4(192, 168, 0, 1))
	vvm, err := Provide(vvmCfg)
	r.NoError(err)

	// Launch VVM1
	problemCtx1 := vvm.Launch(DefaultLeadershipDurationSeconds, DefaultLeadershipAcquisitionDuration)
	r.NoError(problemCtx1.Err(), "VVM1 should start without errors")

	// Get pKey and cCols for leadership key
	pKey := make([]byte, utils.Uint32Size)
	cCols := make([]byte, utils.Uint32Size)
	binary.BigEndian.PutUint32(pKey, 1)
	binary.BigEndian.PutUint32(cCols, uint32(1))

	vvmAppTTLStorage, err := vvm.APIs.IAppStorageProvider.AppStorage(istructs.AppQName_sys_vvm)
	require.NoError(t, err)

	// Leadership key exists
	data := make([]byte, 0)
	ok, err := vvmAppTTLStorage.TTLGet(pKey, cCols, &data)
	r.NoError(err)
	r.True(ok)

	// Shutdown VVM
	problemErr := vvm.Shutdown()
	r.NoError(problemErr)

	// Leadership key doesn't exist
	ok, err = vvmAppTTLStorage.TTLGet(pKey, cCols, &data)
	r.NoError(err)
	r.False(ok)
}

func TestServicePipelineStartFailure(t *testing.T) {
	require := require.New(t)

	vvmCfg := GetTestVVMCfg(net.IPv4(192, 168, 0, 1))
	vvmCfg.VVMPort = -1
	vvm, err := Provide(vvmCfg)
	require.NoError(err)

	problemCtx := vvm.Launch(DefaultLeadershipDurationSeconds, DefaultLeadershipAcquisitionDuration)
	<-problemCtx.Done()

	err = vvm.Shutdown()
	require.Error(err)
	log.Println(err)
}

func TestWrongLaunchAndShutdownUsage(t *testing.T) {
	require := require.New(t)

	vvmCfg := GetTestVVMCfg(net.IPv4(192, 168, 0, 1))
	vvm, err := Provide(vvmCfg)
	require.NoError(err)

	t.Run("panic on Shutdown() if not launched", func(t *testing.T) {
		require.Panics(func() { vvm.Shutdown() })
	})

	t.Run("panic on Launch() if launched already", func(t *testing.T) {
		vvm.Launch(DefaultLeadershipDurationSeconds, DefaultLeadershipAcquisitionDuration)
		defer vvm.Shutdown()
		require.Panics(func() { vvm.Launch(DefaultLeadershipDurationSeconds, DefaultLeadershipAcquisitionDuration) })
	})
}
