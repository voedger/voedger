/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef_test

import (
	"fmt"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
)

func ExampleIAppDefBuilder_AddRate() {

	var app appdef.IAppDef

	// RATE test.rate 10 PER HOUR PER APP PARTITION PER IP

	rateName := appdef.NewQName("test", "rate")
	limitName := appdef.NewQName("test", "limit")

	// how to build AppDef with rates and limits
	{
		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")

		adb.AddRate(rateName, 10, time.Hour, []appdef.RateScope{appdef.RateScope_AppPartition, appdef.RateScope_IP}, "10 times per hour per partition per IP")
		adb.AddLimit(limitName, []appdef.QName{appdef.QNameAnyFunction}, rateName, "limit all commands and queries execution with test.rate")

		app = adb.MustBuild()
	}

	// how to enum rates
	{
		fmt.Println("enum rates:")
		cnt := 0
		for r := range app.Rates {
			cnt++
			fmt.Println("-", cnt, r, fmt.Sprintf("%d per %v per %v", r.Count(), r.Period(), r.Scopes()))
		}
		fmt.Println("overall:", cnt)
	}

	// how to enum limits
	{
		fmt.Println("enum limits:")
		cnt := 0
		for l := range app.Limits {
			cnt++
			fmt.Println("-", cnt, l, fmt.Sprintf("on %v with %v", l.On(), l.Rate()))
		}
		fmt.Println("overall:", cnt)
	}

	// how to find rates and limits
	{
		fmt.Println("find rate:")
		rate := app.Rate(rateName)
		fmt.Println("-", rate, ":", rate.Comment())

		fmt.Println("find limit:")
		limit := app.Limit(limitName)
		fmt.Println("-", limit, ":", limit.Comment())
	}

	// Output:
	// enum rates:
	// - 1 Rate «test.rate» 10 per 1h0m0s per [RateScope_AppPartition RateScope_IP]
	// overall: 1
	// enum limits:
	// - 1 Limit «test.limit» on [sys.AnyFunction] with Rate «test.rate»
	// overall: 1
	// find rate:
	// - Rate «test.rate» : 10 times per hour per partition per IP
	// find limit:
	// - Limit «test.limit» : limit all commands and queries execution with test.rate
}
