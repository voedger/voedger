/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts_test

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts"
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

	report := func(part appparts.IAppPartition, proc appparts.IProc) {
		fmt.Println(part.App(), "partition", part.ID())
		part.AppStructs().AppDef().Types(func(t appdef.IType) {
			if !t.IsSystem() {
				fmt.Println("-", t, t.Comment())
			}
		})
		fmt.Println("- processor:", proc)
	}

	fmt.Println("*** Add ver 1 ***")

	appParts.AddOrReplace(istructs.AppQName_test1_app1, 1, appDef_1_v1, MockProcessors(2, 2, 2))
	appParts.AddOrReplace(istructs.AppQName_test1_app2, 1, appDef_2_v1, MockProcessors(2, 2, 2))

	p1_1, cmd, err := appParts.Borrow(istructs.AppQName_test1_app1, 1, appparts.ProcKind_Command)
	if err != nil {
		panic(err)
	}
	defer p1_1.Release()

	report(p1_1, cmd)

	p2_1, qry, err := appParts.Borrow(istructs.AppQName_test1_app2, 1, appparts.ProcKind_Query)
	if err != nil {
		panic(err)
	}
	defer p2_1.Release()

	report(p2_1, qry)

	fmt.Println("*** Update to ver 2 ***")

	appDef_1_v2 := appDefBuilder("app-1 ver.2")
	appDef_2_v2 := appDefBuilder("app-2 ver.2")
	appConfigs.AddConfig(istructs.AppQName_test1_app1, appDef_1_v2)
	appConfigs.AddConfig(istructs.AppQName_test1_app2, appDef_2_v2)

	appParts.AddOrReplace(istructs.AppQName_test1_app2, 1, appDef_2_v2, MockProcessors(2, 2, 2))
	appParts.AddOrReplace(istructs.AppQName_test1_app1, 1, appDef_1_v2, MockProcessors(2, 2, 2))

	p2_2, prj, err := appParts.Borrow(istructs.AppQName_test1_app2, 1, appparts.ProcKind_Projector)
	if err != nil {
		panic(err)
	}
	defer p2_2.Release()

	report(p2_2, prj)

	p1_2, cmd, err := appParts.Borrow(istructs.AppQName_test1_app1, 1, appparts.ProcKind_Command)
	if err != nil {
		panic(err)
	}
	defer p2_2.Release()

	report(p1_2, cmd)

	// Output:
	// *** Add ver 1 ***
	// test1/app1 partition 1
	// - CDoc «ver.info» app-1 ver.1
	// - processor: Command
	// test1/app2 partition 1
	// - CDoc «ver.info» app-2 ver.1
	// - processor: Query
	// *** Update to ver 2 ***
	// test1/app2 partition 1
	// - CDoc «ver.info» app-2 ver.2
	// - processor: Projector
	// test1/app1 partition 1
	// - CDoc «ver.info» app-1 ver.2
	// - processor: Command
}
