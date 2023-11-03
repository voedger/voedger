/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts_test

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts"
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

	report := func(p appparts.IAppPartition) {
		fmt.Println(p.App(), "partition", p.ID())
		p.AppDef().Types(func(t appdef.IType) {
			if !t.IsSystem() {
				fmt.Println("-", t, t.Comment())
			}
		})
	}

	fmt.Println("*** Add ver 1 ***")

	appParts.AddOrUpdate(istructs.AppQName_test1_app1, 1, appDef("app 1 ver.1"))
	appParts.AddOrUpdate(istructs.AppQName_test1_app2, 1, appDef("app 2 ver.1"))

	p1_1, err := appParts.Borrow(istructs.AppQName_test1_app1, 1)
	if err != nil {
		panic(err)
	}
	defer appParts.Release(p1_1)

	report(p1_1)

	p2_1, err := appParts.Borrow(istructs.AppQName_test1_app2, 1)
	if err != nil {
		panic(err)
	}
	defer appParts.Release(p2_1)

	report(p2_1)

	fmt.Println("*** Update to ver 2 ***")

	appParts.AddOrUpdate(istructs.AppQName_test1_app2, 1, appDef("app 2 ver.2"))
	appParts.AddOrUpdate(istructs.AppQName_test1_app1, 1, appDef("app 1 ver.2"))

	p2_2, err := appParts.Borrow(istructs.AppQName_test1_app2, 1)
	if err != nil {
		panic(err)
	}
	defer appParts.Release(p2_2)

	report(p2_2)

	p1_2, err := appParts.Borrow(istructs.AppQName_test1_app1, 1)
	if err != nil {
		panic(err)
	}
	defer appParts.Release(p1_2)

	report(p1_2)

	// Output:
	// *** Add ver 1 ***
	// test1/app1 partition 1
	// - CDoc «test.doc» app 1 ver.1
	// test1/app2 partition 1
	// - CDoc «test.doc» app 2 ver.1
	// *** Update to ver 2 ***
	// test1/app2 partition 1
	// - CDoc «test.doc» app 2 ver.2
	// test1/app1 partition 1
	// - CDoc «test.doc» app 1 ver.2
}
