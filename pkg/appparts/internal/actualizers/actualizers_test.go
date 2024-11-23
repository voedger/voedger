/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package actualizers

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
	"github.com/voedger/voedger/pkg/istructs"
)

func TestActualizersWaitTimeout(t *testing.T) {
	appName := istructs.AppQName_test1_app1
	partID := istructs.PartitionID(1)
	prjNames := appdef.MustParseQNames("test.p1", "test.p2", "test.p3")

	appDef := func() appdef.IAppDef {
		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")
		wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))
		for _, name := range prjNames {
			wsb.AddProjector(name).SetSync(false).Events().Add(appdef.QNameAnyCommand, appdef.ProjectorEventKind_Execute)
		}
		return adb.MustBuild()
	}

	require := require.New(t)

	t.Run("should ok to wait for all actualizers finished", func(t *testing.T) {
		ctx, stop := context.WithCancel(context.Background())

		actualizers := New(appName, partID)

		app := appDef()

		runCalls := sync.Map{}
		for _, name := range prjNames {
			runCalls.Store(name, 1)
		}
		actualizers.Deploy(ctx, app,
			func(ctx context.Context, app appdef.AppQName, partID istructs.PartitionID, name appdef.QName) {
				require.True(runCalls.CompareAndDelete(name, 1), "actualizer %s was run more than once", name)

				require.Equal(appName, app)
				require.Equal(partID, partID)
				require.Contains(prjNames, name)
				<-ctx.Done()
			})

		require.Equal(prjNames, actualizers.Enum())

		// stop vvm from context, wait actualizers finished
		stop()

		const timeout = 1 * time.Second
		require.True(actualizers.WaitTimeout(timeout))
		require.Empty(actualizers.Enum())

		runCalls.Range(func(key, value any) bool {
			require.Fail("actualizer %s was not run", key.(appdef.QName))
			return true
		})
	})

	t.Run("should timeout to wait infinite run actualizers", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx, stop := context.WithCancel(context.Background())

		actualizers := New(appName, partID)

		app := appDef()

		actualizers.Deploy(ctx, app,
			func(context.Context, appdef.AppQName, istructs.PartitionID, appdef.QName) {
				for {
					time.Sleep(time.Millisecond) // infinite loop
				}
			})

		require.Equal(prjNames, actualizers.Enum())

		// stop vvm from context, wait actualizers finished
		stop()

		const timeout = 1 * time.Second
		require.False(actualizers.WaitTimeout(timeout))
		require.Equal(prjNames, actualizers.Enum())
	})
}
