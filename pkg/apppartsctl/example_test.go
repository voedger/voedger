/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package apppartsctl_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/apppartsctl"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istorageimpl"
	"github.com/voedger/voedger/pkg/istructs"
)

func Example() {
	storage := istorageimpl.Provide(istorage.ProvideMem(), "")

	appParts, cleanupParts, err := appparts.New(storage)
	if err != nil {
		panic(err)
	}
	defer cleanupParts()

	appDef := func(comment ...string) appdef.IAppDef {
		adb := appdef.New()
		adb.AddCDoc(appdef.NewQName("test", "doc")).SetComment(comment...)
		app, err := adb.Build()
		if err != nil {
			panic(err)
		}
		return app
	}

	appPartsCtl, cleanupCtl, err := apppartsctl.New(appParts, []apppartsctl.BuiltInApp{
		{Name: istructs.AppQName_test1_app1,
			Def:      appDef("first app ver.1"),
			NumParts: 7},
		{Name: istructs.AppQName_test1_app2,
			Def:      appDef("second app ver.1"),
			NumParts: 10},
	})

	if err != nil {
		panic(err)
	}
	defer cleanupCtl()

	err = appPartsCtl.Prepare()
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go appPartsCtl.Run(ctx)

	borrowAndRelease := func(appName istructs.AppQName, partID istructs.PartitionID) {
		part, err := appParts.Borrow(appName, partID)
		for errors.Is(err, appparts.ErrNotFound) {
			time.Sleep(time.Nanosecond)
			part, err = appParts.Borrow(appName, partID) // Service lag, retry until found
		}
		if err != nil {
			panic(err)
		}

		defer appParts.Release(part)

		fmt.Println(part.App(), "part", part.ID())
		part.AppDef().Types(
			func(typ appdef.IType) {
				if !typ.IsSystem() {
					fmt.Println("-", typ, typ.Comment())
				}
			})
	}

	borrowAndRelease(istructs.AppQName_test1_app1, 1)
	appParts.AddOrUpdate(istructs.AppQName_test1_app1, 1, appDef("first app ver.2"))
	borrowAndRelease(istructs.AppQName_test1_app1, 1)

	borrowAndRelease(istructs.AppQName_test1_app2, 2)
	appParts.AddOrUpdate(istructs.AppQName_test1_app2, 2, appDef("second app ver.2"))
	borrowAndRelease(istructs.AppQName_test1_app2, 2)

	cancel()

	// Output:
	// test1/app1 part 1
	// - CDoc «test.doc» first app ver.1
	// test1/app1 part 1
	// - CDoc «test.doc» first app ver.2
	// test1/app2 part 2
	// - CDoc «test.doc» second app ver.1
	// test1/app2 part 2
	// - CDoc «test.doc» second app ver.2
}
