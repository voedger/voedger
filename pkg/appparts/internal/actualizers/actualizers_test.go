/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package actualizers_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/appdef/filter"
	"github.com/voedger/voedger/pkg/appparts/internal/actualizers"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
	"github.com/voedger/voedger/pkg/istructs"
)

func TestActualizersWaitTimeout(t *testing.T) {
	appName := istructs.AppQName_test1_app1
	initialPartID := istructs.PartitionID(1)
	wsName := appdef.NewQName("test", "workspace")
	prjNames := appdef.MustParseQNames("test.p1", "test.p2", "test.p3")

	appDef := func() appdef.IAppDef {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")
		wsb := adb.AddWorkspace(wsName)
		_ = wsb.AddCommand(appdef.NewQName("test", "command"))
		for _, name := range prjNames {
			prj := wsb.AddProjector(name)
			prj.Events().Add(
				[]appdef.OperationKind{appdef.OperationKind_Execute},
				filter.WSTypes(wsName, appdef.TypeKind_Command))
			prj.SetSync(false)
		}
		return adb.MustBuild()
	}

	require := require.New(t)

	t.Run("should ok to wait for all actualizers finished", func(t *testing.T) {
		ctx, stop := context.WithCancel(context.Background())

		actualizers := actualizers.New(appName, initialPartID)

		app := appDef()

		runCalls := sync.Map{}
		for _, name := range prjNames {
			runCalls.Store(name, 1)
		}
		actualizers.Deploy(ctx, app,
			func(ctx context.Context, app appdef.AppQName, deployingPartID istructs.PartitionID, name appdef.QName) {
				require.True(runCalls.CompareAndDelete(name, 1), "actualizer %s was run more than once", name)

				require.Equal(appName, app)
				require.Equal(initialPartID, deployingPartID)
				require.Contains(prjNames, name)
				<-ctx.Done()
			})

		require.Equal(prjNames, actualizers.Enum())

		// stop vvm from context, wait actualizers finished
		stop()

		const timeout = 1 * time.Second
		require.True(waitFor(actualizers, timeout))
		require.Empty(actualizers.Enum())

		runCalls.Range(func(key, value any) bool {
			require.Fail(fmt.Sprintf("actualizer %s was not run", key.(appdef.QName)))
			return true
		})
	})

	t.Run("should timeout to wait infinite run actualizers", func(t *testing.T) {

		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}

		ctx, stop := context.WithCancel(context.Background())

		actualizers := actualizers.New(appName, initialPartID)

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
		require.False(waitFor(actualizers, timeout))
		require.Equal(prjNames, actualizers.Enum())
	})
}

func waitFor(pa *actualizers.PartitionActualizers, timeout time.Duration) bool {
	done := make(chan struct{})
	go func() {
		pa.Wait()
		close(done)
	}()

	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
}
