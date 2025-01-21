/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef_test

import (
	"fmt"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/appdef/filter"
)

func ExampleRates() {

	var app appdef.IAppDef

	// RATE test.rate 10 PER HOUR PER APP PARTITION PER IP

	wsName := appdef.NewQName("test", "workspace")
	rateAllName, rateEachName := appdef.NewQName("test", "rateAll"), appdef.NewQName("test", "rateEach")
	limitAllName, limitEachName := appdef.NewQName("test", "limitAll"), appdef.NewQName("test", "limitEach")
	cmdName, queryName := appdef.NewQName("test", "command"), appdef.NewQName("test", "query")

	// how to build AppDef with rates and limits
	{
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		_ = wsb.AddCommand(cmdName)
		_ = wsb.AddQuery(queryName)

		wsb.AddRate(rateAllName, 10, time.Hour, []appdef.RateScope{appdef.RateScope_AppPartition, appdef.RateScope_IP}, "10 times per hour per partition per IP")
		wsb.AddRate(rateEachName, 1, 10*time.Minute, []appdef.RateScope{appdef.RateScope_AppPartition, appdef.RateScope_IP}, "1 times per 10 minutes per partition per IP")

		wsb.AddLimit(limitAllName, []appdef.OperationKind{appdef.OperationKind_Execute}, appdef.LimitFilterOption_ALL, filter.AllWSFunctions(wsName), rateAllName, "limit all commands and queries execution with test.rateAll")
		wsb.AddLimit(limitEachName, []appdef.OperationKind{appdef.OperationKind_Execute}, appdef.LimitFilterOption_EACH, filter.AllWSFunctions(wsName), rateEachName, "limit each command and query execution with test.rateEach")

		app = adb.MustBuild()
	}

	// how to enum rates
	{
		fmt.Println("enum rates:")
		cnt := 0
		for r := range appdef.Rates(app.Types()) {
			cnt++
			fmt.Println("-", cnt, r, fmt.Sprintf("%d per %v per %v", r.Count(), r.Period(), r.Scopes()))
		}
		fmt.Println("overall:", cnt)
	}

	// how to enum limits
	{
		fmt.Println("enum limits:")
		cnt := 0
		for l := range appdef.Limits(app.Types()) {
			cnt++
			fmt.Println("-", cnt, l, fmt.Sprintf("%v ON %v BY %v", l.Ops(), l.Filter(), l.Rate()))
		}
		fmt.Println("overall:", cnt)
	}

	// how to find rates and limits
	{
		fmt.Println("find rate:")
		rate := appdef.Rate(app.Type, rateAllName)
		fmt.Println("-", rate, ":", rate.Comment())

		fmt.Println("find limit:")
		limit := appdef.Limit(app.Type, limitAllName)
		fmt.Println("-", limit, ":", limit.Comment())
	}

	// Output:
	// enum rates:
	// - 1 Rate «test.rateAll» 10 per 1h0m0s per [RateScope_AppPartition RateScope_IP]
	// - 2 Rate «test.rateEach» 1 per 10m0s per [RateScope_AppPartition RateScope_IP]
	// overall: 2
	// enum limits:
	// - 1 Limit «test.limitAll» [OperationKind_Execute] ON ALL FUNCTIONS FROM test.workspace BY Rate «test.rateAll»
	// - 2 Limit «test.limitEach» [OperationKind_Execute] ON EACH FUNCTIONS FROM test.workspace BY Rate «test.rateEach»
	// overall: 2
	// find rate:
	// - Rate «test.rateAll» : 10 times per hour per partition per IP
	// find limit:
	// - Limit «test.limitAll» : limit all commands and queries execution with test.rateAll
}
