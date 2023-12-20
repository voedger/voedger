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
	"github.com/voedger/voedger/pkg/cluster"
	"github.com/voedger/voedger/pkg/iratesce"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istorageimpl"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/itokensjwt"
)

func Example() {
	appDefBuilder := func(verInfo ...string) appdef.IAppDefBuilder {
		adb := appdef.New()
		adb.AddCDoc(appdef.NewQName("ver", "info")).SetComment(verInfo...)
		return adb
	}

	appConfigs := istructsmem.AppConfigsType{}
	appDef_1_v1 := appDefBuilder("app-1 ver.1")
	appDef_2_v1 := appDefBuilder("app-2 ver.1")
	appConfigs.AddConfig(istructs.AppQName_test1_app1, appDef_1_v1)
	appConfigs.AddConfig(istructs.AppQName_test1_app2, appDef_2_v1)

	appStructs := istructsmem.Provide(
		appConfigs,
		iratesce.TestBucketsFactory,
		payloads.TestAppTokensFactory(itokensjwt.TestTokensJWT()),
		istorageimpl.Provide(istorage.ProvideMem(), ""))

	appParts, cleanupParts, err := appparts.New(appStructs)
	if err != nil {
		panic(err)
	}
	defer cleanupParts()

	appPartsCtl, cleanupCtl, err := apppartsctl.New(appParts, []apppartsctl.BuiltInApp{
		{Name: istructs.AppQName_test1_app1,
			Def:            appDef_1_v1,
			PartsCount:     2,
			EnginePoolSize: [cluster.ProcessorKind_Count]int{2, 2, 2}},
		{Name: istructs.AppQName_test1_app2,
			Def:            appDef_2_v1,
			PartsCount:     3,
			EnginePoolSize: [cluster.ProcessorKind_Count]int{2, 2, 2}},
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

	borrow_work_release := func(appName istructs.AppQName, partID istructs.PartitionID, proc cluster.ProcessorKind) {
		part, err := appParts.Borrow(appName, partID, proc)
		for errors.Is(err, appparts.ErrNotFound) {
			time.Sleep(time.Nanosecond)
			part, err = appParts.Borrow(appName, partID, proc) // Service lag, retry until found
		}
		if err != nil {
			panic(err)
		}

		defer part.Release()

		fmt.Println(part.App(), "part", part.ID())
		part.AppStructs().AppDef().Types(
			func(typ appdef.IType) {
				if !typ.IsSystem() {
					fmt.Println("-", typ, typ.Comment())
				}
			})
	}

	borrow_work_release(istructs.AppQName_test1_app1, 1, cluster.ProcessorKind_Command)
	borrow_work_release(istructs.AppQName_test1_app2, 1, cluster.ProcessorKind_Query)

	cancel()

	// Output:
	// test1/app1 part 1
	// - CDoc «ver.info» app-1 ver.1
	// test1/app2 part 1
	// - CDoc «ver.info» app-2 ver.1
}
