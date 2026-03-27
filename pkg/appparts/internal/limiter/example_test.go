/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package limiter_test

import (
	"fmt"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/appdef/filter"
	"github.com/voedger/voedger/pkg/appparts/internal/limiter"
	"github.com/voedger/voedger/pkg/goutils/testingu"
	"github.com/voedger/voedger/pkg/iratesce"
	"github.com/voedger/voedger/pkg/istructs"
)

func Example() {
	cmd1Name := appdef.NewQName("test", "cmd1")
	cmd2Name := appdef.NewQName("test", "cmd2")
	app := func() appdef.IAppDef {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsName := appdef.NewQName("test", "workspace")
		wsb := adb.AddWorkspace(wsName)
		_ = wsb.AddCommand(cmd1Name)
		_ = wsb.AddCommand(cmd2Name)

		// Add rate and limit for workspace: 5 commands per minute
		wsRateName := appdef.NewQName("test", "wsRate")
		wsb.AddRate(wsRateName, 5, time.Minute, []appdef.RateScope{appdef.RateScope_Workspace})
		wsb.AddLimit(
			appdef.NewQName("test", "wsLimit"),
			[]appdef.OperationKind{appdef.OperationKind_Execute},
			appdef.LimitFilterOption_ALL,
			filter.WSTypes(wsName, appdef.TypeKind_Command),
			wsRateName)

		// Add rate and limit for IP: 3 commands per minute for each IP address for each command
		ipRateName := appdef.NewQName("test", "ipRate")
		wsb.AddRate(ipRateName, 3, time.Minute, []appdef.RateScope{appdef.RateScope_IP})
		wsb.AddLimit(
			appdef.NewQName("test", "ipLimit"),
			[]appdef.OperationKind{appdef.OperationKind_Execute},
			appdef.LimitFilterOption_EACH,
			filter.WSTypes(wsName, appdef.TypeKind_Command),
			ipRateName)

		return adb.MustBuild()
	}()

	buckets := iratesce.TestBucketsFactory()

	Limiter := limiter.New(app, buckets)

	// Check limits for cmd1
	fmt.Println(Limiter.Exceeded(cmd1Name, appdef.OperationKind_Execute, 1, `addr1`))
	fmt.Println(Limiter.Exceeded(cmd1Name, appdef.OperationKind_Execute, 1, `addr1`))
	fmt.Println(Limiter.Exceeded(cmd1Name, appdef.OperationKind_Execute, 1, `addr1`))

	fmt.Println(Limiter.Exceeded(cmd1Name, appdef.OperationKind_Execute, 1, `addr1`)) // Exceeded IP

	// Check limits for cmd2
	fmt.Println(Limiter.Exceeded(cmd2Name, appdef.OperationKind_Execute, 1, `addr1`))
	fmt.Println(Limiter.Exceeded(cmd2Name, appdef.OperationKind_Execute, 1, `addr1`))

	fmt.Println(Limiter.Exceeded(cmd2Name, appdef.OperationKind_Execute, 1, `addr1`)) // Exceeded WS

	testingu.MockTime.Add(time.Minute) // Wait for the next minute

	// check limits is respawned
	fmt.Println(Limiter.Exceeded(cmd1Name, appdef.OperationKind_Execute, 1, `addr1`))
	fmt.Println(Limiter.Exceeded(cmd2Name, appdef.OperationKind_Execute, 1, `addr1`))

	// check limits for unlimited command
	fmt.Println(Limiter.Exceeded(appdef.NewQName("test", "cmd3"), appdef.OperationKind_Execute, 1, `addr1`))
	fmt.Println(Limiter.Exceeded(appdef.NewQName("test", "cmd3"), appdef.OperationKind_Execute, 1, `addr1`))
	fmt.Println(Limiter.Exceeded(appdef.NewQName("test", "cmd3"), appdef.OperationKind_Execute, 1, `addr1`))
	fmt.Println(Limiter.Exceeded(appdef.NewQName("test", "cmd3"), appdef.OperationKind_Execute, 1, `addr1`))

	// Output:
	// false .
	// false .
	// false .
	// true test.ipLimit
	// false .
	// false .
	// true test.wsLimit
	// false .
	// false .
	// false .
	// false .
	// false .
	// false .
}

func Example_resetLimits() {
	cmdName := appdef.NewQName("test", "cmd")
	app := func() appdef.IAppDef {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsName := appdef.NewQName("test", "workspace")
		wsb := adb.AddWorkspace(wsName)
		_ = wsb.AddCommand(cmdName)

		rateName := appdef.NewQName("test", "rate")
		wsb.AddRate(rateName, 3, time.Minute, []appdef.RateScope{appdef.RateScope_Workspace})
		wsb.AddLimit(
			appdef.NewQName("test", "limit"),
			[]appdef.OperationKind{appdef.OperationKind_Execute},
			appdef.LimitFilterOption_EACH,
			filter.WSTypes(wsName, appdef.TypeKind_Command),
			rateName)

		return adb.MustBuild()
	}()

	buckets := iratesce.TestBucketsFactory()
	Limiter := limiter.New(app, buckets)

	var ws istructs.WSID = 1

	fmt.Println(Limiter.Exceeded(cmdName, appdef.OperationKind_Execute, ws, ``))
	fmt.Println(Limiter.Exceeded(cmdName, appdef.OperationKind_Execute, ws, ``))
	fmt.Println(Limiter.Exceeded(cmdName, appdef.OperationKind_Execute, ws, ``))
	fmt.Println(Limiter.Exceeded(cmdName, appdef.OperationKind_Execute, ws, ``)) // exceeded

	Limiter.ResetLimits(cmdName, appdef.OperationKind_Execute, ws, ``)

	fmt.Println(Limiter.Exceeded(cmdName, appdef.OperationKind_Execute, ws, ``)) // allowed again

	// Output:
	// false .
	// false .
	// false .
	// true test.limit
	// false .
}
