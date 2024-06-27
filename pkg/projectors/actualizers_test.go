/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package projectors

import (
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
)

func Test_actualizers_DeployPartition(t *testing.T) {
	require := require.New(t)

	appName := istructs.AppQName_test1_app1

	const (
		partCount = 10
		prjCount  = 10
		evCount   = 10
	)

	prjName := func(i int) appdef.QName {
		return appdef.NewQName("test", fmt.Sprintf("prj_%d", i))
	}

	var (
		counter int64
		finish  int64 = partCount * prjCount * evCount
	)

	appParts, actualizers, appStructs, start, stop := deployTestAppEx(
		appName, partCount, false,
		func(appDef appdef.IAppDefBuilder) {
			appDef.AddPackage("test", "test.com/test")
			appDef.AddCommand(testQName)
			for i := 0; i < prjCount; i++ {
				appDef.AddProjector(prjName(i)).Events().Add(testQName, appdef.ProjectorEventKind_Execute)
			}
			addWS(appDef, testWorkspace, testWorkspaceDescriptor)
		},
		func(cfg *istructsmem.AppConfigType) {
			cfg.Resources.Add(istructsmem.NewCommandFunction(testQName, istructsmem.NullCommandExec))
			for i := 0; i < prjCount; i++ {
				cfg.AddAsyncProjectors(istructs.Projector{
					Name: prjName(i),
					Func: func(istructs.IPLogEvent, istructs.IState, istructs.IIntents) error {
						atomic.AddInt64(&counter, 1)
						return nil
					},
				})
			}
		},
		&BasicAsyncActualizerConfig{})

	testWS := istructs.WSID(1001)

	ofs := istructs.Offset(1)
	createWS(appStructs, testWS, testWorkspaceDescriptor, 0, ofs)
	ofs++

	f := pLogFiller{
		app:      appStructs,
		cmdQName: testQName,
	}
	for i := 0; i < evCount; i++ {
		for p := istructs.PartitionID(0); p < partCount; p++ {
			f.partition, f.offset = p, ofs
			f.fill(testWS)
			ofs++
		}
	}

	appParts.DeployAppPartitions(appName,
		func() []istructs.PartitionID {
			pp := make([]istructs.PartitionID, partCount)
			for p := istructs.PartitionID(0); p < partCount; p++ {
				pp[int(p)] = p
			}
			return pp
		}())

	start()

	// Wait for the projectors

	for atomic.LoadInt64(&counter) < finish {
		time.Sleep(time.Millisecond)
	}

	// Check the actualizers internal getMetrics
	getMetrics := actualizers.(interface{ metrics() actualizersMetrics }).metrics

	t.Run("test actualizers internal metrics after deploy", func(t *testing.T) {
		m := getMetrics()
		require.Len(m.apps, 1)
		require.Contains(m.apps, appName)

		require.Len(m.parts[appName], partCount)
		require.Len(m.actualizers[appName], partCount)
		for p := istructs.PartitionID(0); p < partCount; p++ {
			require.Contains(m.parts[appName], p)
			require.Contains(m.actualizers[appName], p)
		}

		for _, id := range m.parts[appName] {
			p := m.actualizers[appName][id]
			require.Len(p, prjCount)
			for i := 0; i < prjCount; i++ {
				require.Contains(p, prjName(i))
			}
		}
	})

	t.Run("Should be panic if reassign IAppPartitions", func(t *testing.T) {
		require.Panics(
			func() { actualizers.SetAppPartitions(nil) },
			require.Is(errors.ErrUnsupported),
			require.Has("unable to reset"))
	})

	t.Run("Should be error if deploy unknown application", func(t *testing.T) {
		require.Error(
			actualizers.DeployPartition(appdef.NewAppQName("test", "unknown"), 0),
			require.Is(appparts.ErrNotFound),
			require.Has("test/unknown"))
	})

	t.Run("Should be ok to undeploy partitions", func(t *testing.T) {

		t.Run("Should be ok undeploy unknown application or unknown partition", func(t *testing.T) {
			require.NotPanics(func() {
				actualizers.UndeployPartition(appdef.NewAppQName("test", "unknown"), 0)
				actualizers.UndeployPartition(appName, partCount+1)
			})
		})

		// undeploy the even partitions
		for p := istructs.PartitionID(0); p < partCount; p++ {
			if p%2 == 0 {
				actualizers.UndeployPartition(appName, p)
			}
		}

		// wait for undeploy finished
		m := getMetrics()
		for len(m.parts[appName]) > partCount/2 {
			time.Sleep(time.Millisecond)
			m = getMetrics()
		}

		t.Run("test actualizers internal metrics after undeploy", func(t *testing.T) {
			m := actualizers.(interface{ metrics() actualizersMetrics }).metrics()
			require.Len(m.apps, 1)
			require.Contains(m.apps, appName)

			require.Len(m.parts[appName], partCount/2)
			require.Len(m.actualizers[appName], partCount/2)
			for p := istructs.PartitionID(0); p < partCount; p++ {
				if p%2 == 0 {
					require.NotContains(m.parts[appName], p)
					require.NotContains(m.actualizers[appName], p)
				} else {
					require.Contains(m.parts[appName], p)
					require.Contains(m.actualizers[appName], p)
				}
			}

			for _, id := range m.parts[appName] {
				p := m.actualizers[appName][id]
				require.Len(p, prjCount)
				for i := 0; i < prjCount; i++ {
					require.Contains(p, prjName(i))
				}
			}
		})
	})

	// stop services
	stop()

	require.Equal(finish, counter)
}
