/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */
package vvm

import (
	"context"
	"encoding/binary"
	"net"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/coreutils/utils"
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istorage/mem"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/parser"
	"github.com/voedger/voedger/pkg/sys/smtp"
	"github.com/voedger/voedger/pkg/sys/sysprovide"
	builtinapps "github.com/voedger/voedger/pkg/vvm/builtin"
	"github.com/voedger/voedger/pkg/vvm/builtin/clusterapp"
	"github.com/voedger/voedger/pkg/vvm/builtin/registryapp"
	"github.com/voedger/voedger/pkg/vvm/storage"
)

func getMemIAppStorage(t *testing.T, iTime coreutils.ITime) istorage.IAppStorage {
	sf := mem.Provide(iTime)
	san, err := istorage.NewSafeAppName(
		istructs.AppQName_sys_cluster,
		func(name string) (bool, error) {
			return true, nil
		},
	)
	require.NoError(t, err)

	err = sf.Init(san)
	require.NoError(t, err)

	storage, err := sf.AppStorage(san)
	require.NoError(t, err)

	return storage
}

func patchVVM(vvm *VoedgerVM, vvmTTLStorage storage.IVVMAppTTLStorage) {
	vvm.VVMAppTTLStorage = vvmTTLStorage
}

func getTestVVMCfg(clusterSize ClusterSize, ip net.IP, iTime coreutils.ITime) *VVMConfig {
	vvmCfg1 := NewVVMDefaultConfig()
	vvmCfg1.IP = ip
	vvmCfg1.ClusterSize = clusterSize
	vvmCfg1.VVMPort = 0
	vvmCfg1.MetricsServicePort = 0
	vvmCfg1.Time = iTime
	vvmCfg1.VVMAppsBuilder.Add(istructs.AppQName_test1_app1, func(apis builtinapps.APIs, cfg *istructsmem.AppConfigType, ep extensionpoints.IExtensionPoint) builtinapps.Def {
		sysPackageFS := sysprovide.Provide(cfg)
		return builtinapps.Def{
			AppDeploymentDescriptor: appparts.AppDeploymentDescriptor{
				NumParts:         10,
				EnginePoolSize:   appparts.PoolSize(10, 10, 20, 10),
				NumAppWorkspaces: istructs.DefaultNumAppWorkspaces,
			},
			AppQName: istructs.AppQName_test1_app1,
			Packages: []parser.PackageFS{{
				Path: "github.com/voedger/voedger/pkg/app1",
				FS: fstest.MapFS{
					"app.vsql": &fstest.MapFile{
						Data: []byte(`
APPLICATION app1();

ALTERABLE WORKSPACE test_wsWS (

	DESCRIPTOR test_ws (
		IntFld int32 NOT NULL,
		StrFld varchar(1024)
	);
);`),
					},
				},
			}, sysPackageFS},
		}
	})
	vvmCfg1.VVMAppsBuilder.Add(istructs.AppQName_sys_registry, registryapp.Provide(smtp.Cfg{}, 10))
	vvmCfg1.VVMAppsBuilder.Add(istructs.AppQName_sys_cluster, clusterapp.Provide())

	return &vvmCfg1
}

func TestBasic(t *testing.T) {
	t.Run("VVMStartAndStop", func(t *testing.T) {
		r := require.New(t)

		vvmCfg1 := getTestVVMCfg(1, net.IPv4(192, 168, 0, 1), coreutils.MockTime)
		vvm1, err := ProvideVVM(vvmCfg1)
		r.NoError(err)
		r.NotNil(vvm1)

		// Launch VVM1
		problemCtx1 := vvm1.LaunchNew(5 * time.Second)
		r.NoError(problemCtx1.Err(), "VVM1 should start without errors")
		r.NoError(vvm1.ShutdownNew())
		r.ErrorIs(vvm1.shutdownedCtx.Err(), context.Canceled)
	})

	t.Run("LeadershipCollision", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		r := require.New(t)

		iTime := coreutils.MockTime
		memStorage := getMemIAppStorage(t, iTime)

		vvmCfg1 := getTestVVMCfg(1, net.IPv4(192, 168, 0, 1), iTime)
		vvm1, err := ProvideVVM(vvmCfg1)
		r.NoError(err)
		r.NotNil(vvm1)
		// patch VVM1 with memStorage
		patchVVM(vvm1, memStorage)

		vvmCfg2 := getTestVVMCfg(1, net.IPv4(192, 168, 0, 2), iTime)

		duration := time.Second
		// Launch VVM1
		problemCtx1 := vvm1.LaunchNew(4 * duration)
		r.NotNil(problemCtx1, "VVM1 should start without errors")
		r.NoError(problemCtx1.Err(), "VVM1 should start without errors")

		// Launch VVM2, expecting leadership acquisition to fail
		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()

			vvm2, err := ProvideVVM(vvmCfg2)
			r.NoError(err)
			r.NotNil(vvm2)
			// patch VVM2 with the same memStorage as VVM1
			patchVVM(vvm2, memStorage)

			// goroutine for ticking time after VVM2 starts leadership acquisition
			go func() {
				<-vvm2.startedLeadershipAcquisition
				iTime.Sleep(duration)
			}()
			problemCtx2 := vvm2.LaunchNew(duration)

			<-problemCtx2.Done()

			r.Error(problemCtx2.Err(), "VVM2 should not acquire leadership")
			r.ErrorIs(vvm2.ShutdownNew(), ErrVVMLeadershipAcquisition)
		}()

		wg.Wait()
		r.NoError(vvm1.ShutdownNew())
	})
}

func TestAutomaticShutdownOnLeadershipLost(t *testing.T) {
	r := require.New(t)

	vvmCfg1 := getTestVVMCfg(
		1,
		net.IPv4(192, 168, 0, 1),
		coreutils.MockTime,
	)
	vvm1, err := ProvideVVM(vvmCfg1)
	r.NoError(err)
	r.NotNil(vvm1)

	duration := 5 * time.Second
	// Launch VVM1
	problemCtx1 := vvm1.LaunchNew(2 * duration)
	r.NoError(problemCtx1.Err(), "VVM1 should start without errors")

	// Simulate leadership loss
	pKey := make([]byte, utils.Uint32Size)
	cCols := make([]byte, utils.Uint32Size)
	binary.BigEndian.PutUint32(pKey, 1)
	binary.BigEndian.PutUint32(cCols, uint32(1))

	ok, err := vvm1.VVMAppTTLStorage.CompareAndSwap(
		pKey,
		cCols,
		[]byte(vvmCfg1.IP.String()),
		[]byte("another_value"),
		50,
	)
	r.NoError(err)
	r.True(ok)

	// Bump mock time
	coreutils.MockTime.Sleep(duration)
	// Check problem context
	<-problemCtx1.Done()
	r.ErrorIs(vvm1.ShutdownNew(), ErrLeadershipLost)
}

func TestCancelLeadershipOnManualShutdown(t *testing.T) {
	r := require.New(t)

	vvmCfg1 := getTestVVMCfg(
		1,
		net.IPv4(192, 168, 0, 1),
		coreutils.MockTime,
	)
	vvm1, err := ProvideVVM(vvmCfg1)
	r.NoError(err)
	r.NotNil(vvm1)

	duration := 5 * time.Second
	// Launch VVM1
	problemCtx1 := vvm1.LaunchNew(duration)
	r.NoError(problemCtx1.Err(), "VVM1 should start without errors")

	// Get pKey and cCols for leadership key
	pKey := make([]byte, utils.Uint32Size)
	cCols := make([]byte, utils.Uint32Size)
	binary.BigEndian.PutUint32(pKey, 1)
	binary.BigEndian.PutUint32(cCols, uint32(1))

	// Leadership key exists
	data := make([]byte, 0)
	ok, err := vvm1.VVMAppTTLStorage.(istorage.IAppStorage).Get(pKey, cCols, &data)
	r.NoError(err)
	r.True(ok)
	// Shutdown VVM1
	problemErr := vvm1.ShutdownNew()
	r.NoError(problemErr)

	// Leadership key doesn't exist
	ok, err = vvm1.VVMAppTTLStorage.(istorage.IAppStorage).Get(pKey, cCols, &data)
	r.NoError(err)
	r.False(ok)
}
