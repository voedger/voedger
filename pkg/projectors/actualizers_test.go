/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package projectors

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
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

	appParts, appStructs, start, stop := deployTestApp(
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

	// stop services
	stop()

	require.Equal(finish, counter)
}
