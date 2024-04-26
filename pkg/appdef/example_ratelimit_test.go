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

	// how to build AppDef with rates
	{
		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")

		adb.AddRate(rateName, 10, time.Hour, []appdef.RateScope{appdef.RateScope_AppPartition, appdef.RateScope_IP}, "10 times per hour per partition per IP")

		app = adb.MustBuild()
	}

	// how to enum rates
	{
		cnt := 0
		app.Rates(func(r appdef.IRate) {
			cnt++
			fmt.Println(cnt, r, fmt.Sprintf("%d per %v per %v", r.Count(), r.Period(), r.Scopes()))
		})
		fmt.Println("overall:", cnt)
	}

	// how to inspect builded AppDef with rates
	{
		rate := app.Rate(rateName)
		fmt.Println(rate, ":", rate.Comment())
	}

	// Output:
	// 1 Rate «test.rate» 10 per 1h0m0s per [AppPartition IP]
	// overall: 1
	// Rate «test.rate» : 10 times per hour per partition per IP
}
