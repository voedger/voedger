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
	"testing/fstest"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/coreutils/utils"
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/parser"
	"github.com/voedger/voedger/pkg/sys/smtp"
	"github.com/voedger/voedger/pkg/sys/sysprovide"
	builtinapps "github.com/voedger/voedger/pkg/vvm/builtin"
	"github.com/voedger/voedger/pkg/vvm/builtin/clusterapp"
	"github.com/voedger/voedger/pkg/vvm/builtin/registryapp"
)

func TestBasic(t *testing.T) {
	t.Run("VVMStartAndStop", func(t *testing.T) {
		r := require.New(t)

		vvmCfg1 := getTestVVMCfg(net.IPv4(192, 168, 0, 1))
		vvm1, err := Provide(vvmCfg1)
		r.NoError(err)

		// Launch VVM1
		problemCtx := vvm1.Launch(DefaultLeadershipDurationSeconds, DefaultLeadershipAcquisitionDuration)
		r.NoError(problemCtx.Err())
		r.NoError(vvm1.Shutdown())
		<-vvm1.shutdownedCtx.Done()
	})

	t.Run("LeadershipCollision", func(t *testing.T) {
		r := require.New(t)

		iTime := coreutils.MockTime
		vvmCfg1 := getTestVVMCfg(net.IPv4(192, 168, 0, 1))

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

			vvmCfg2 := getTestVVMCfg(net.IPv4(192, 168, 0, 2))

			// set vvmCfg2 storage factory to the one from vvm1
			vvmCfg2.StorageFactory = func() (provider istorage.IAppStorageFactory, err error) {
				return sharedStorageFactory, nil
			}
			vvmCfg2.KeyspaceNameSuffix = suffix

			vvm2, err := Provide(vvmCfg2)
			r.NoError(err)

			go func() {
				// force case <-leadershipAcquistionTimerCh to fire in tryToAcquireLeadership()
				<-vvm2.leadershipAcquisitionTimeArmed
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

func TestAutomaticShutdownOnLeadershipLost(t *testing.T) {
	r := require.New(t)

	vvmCfg := getTestVVMCfg(net.IPv4(192, 168, 0, 1))
	vvm1, err := Provide(vvmCfg)
	r.NoError(err)

	// Launch VVM1
	problemCtx := vvm1.Launch(DefaultLeadershipDurationSeconds, DefaultLeadershipAcquisitionDuration)
	r.NoError(problemCtx.Err())

	// Simulate leadership loss
	pKey := make([]byte, utils.Uint32Size)
	cCols := make([]byte, utils.Uint32Size)
	binary.BigEndian.PutUint32(pKey, 1)
	binary.BigEndian.PutUint32(cCols, uint32(1))

	vvmAppTTLStorage, err := vvm1.APIs.IAppStorageProvider.AppStorage(appdef.NewAppQName(istructs.SysOwner, "vvm"))
	require.NoError(t, err)
	ok, err := vvmAppTTLStorage.CompareAndSwap(
		pKey,
		cCols,
		[]byte(vvmCfg.IP.String()),
		[]byte("another_value"),
		50,
	)
	r.NoError(err)
	r.True(ok)

	// Bump mock time
	coreutils.MockTime.Sleep(time.Duration(DefaultLeadershipDurationSeconds) * time.Second)

	// Check problem context
	<-problemCtx.Done()
	r.ErrorIs(vvm1.Shutdown(), ErrLeadershipLost)
}

func TestCancelLeadershipOnManualShutdown(t *testing.T) {
	r := require.New(t)

	vvmCfg := getTestVVMCfg(net.IPv4(192, 168, 0, 1))
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

	vvmAppTTLStorage, err := vvm.APIs.IAppStorageProvider.AppStorage(appdef.NewAppQName(istructs.SysOwner, "vvm"))
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

	vvmCfg := getTestVVMCfg(net.IPv4(192, 168, 0, 1))
	vvmCfg.VVMPort = -1
	vvm, err := Provide(vvmCfg)
	require.NoError(err)

	problemCtx := vvm.Launch(DefaultLeadershipDurationSeconds, DefaultLeadershipAcquisitionDuration)
	<-problemCtx.Done()

	err = vvm.Shutdown()
	require.Error(err)
	log.Println(err)
}

func getTestVVMCfg(ip net.IP) *VVMConfig {
	vvmCfg := NewVVMDefaultConfig()
	vvmCfg.VVMPort = 0
	vvmCfg.IP = ip
	vvmCfg.MetricsServicePort = 0
	vvmCfg.Time = coreutils.MockTime
	vvmCfg.VVMAppsBuilder.Add(istructs.AppQName_test1_app1, func(apis builtinapps.APIs, cfg *istructsmem.AppConfigType, ep extensionpoints.IExtensionPoint) builtinapps.Def {
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
	vvmCfg.VVMAppsBuilder.Add(istructs.AppQName_sys_registry, registryapp.Provide(smtp.Cfg{}, 10))
	vvmCfg.VVMAppsBuilder.Add(istructs.AppQName_sys_cluster, clusterapp.Provide())

	return &vvmCfg
}
