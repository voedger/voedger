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

	report := func(part appparts.IAppPartition) {
		fmt.Println(part.App(), "partition", part.ID())
		part.AppStructs().AppDef().Types(func(t appdef.IType) {
			if !t.IsSystem() {
				fmt.Println("-", t, t.Comment())
			}
		})
	}

	fmt.Println("*** Add ver 1 ***")

	appParts.DeployApp(istructs.AppQName_test1_app1, appDef_1_v1, MockEngines(2, 2, 2))
	appParts.DeployApp(istructs.AppQName_test1_app2, appDef_2_v1, MockEngines(2, 2, 2))

	appParts.DeployAppPartitions(istructs.AppQName_test1_app1, []istructs.PartitionID{1})
	appParts.DeployAppPartitions(istructs.AppQName_test1_app2, []istructs.PartitionID{1})

	a1_v1_p1, err := appParts.Borrow(istructs.AppQName_test1_app1, 1, appparts.ProcKind_Command)
	if err != nil {
		panic(err)
	}
	defer a1_v1_p1.Release()

	report(a1_v1_p1)

	a2_v1_p1, err := appParts.Borrow(istructs.AppQName_test1_app2, 1, appparts.ProcKind_Query)
	if err != nil {
		panic(err)
	}
	defer a2_v1_p1.Release()

	report(a2_v1_p1)

	fmt.Println("*** Update to ver 2 ***")

	appDef_1_v2 := appDefBuilder("app-1 ver.2")
	appDef_2_v2 := appDefBuilder("app-2 ver.2")
	appConfigs.AddConfig(istructs.AppQName_test1_app1, appDef_1_v2)
	appConfigs.AddConfig(istructs.AppQName_test1_app2, appDef_2_v2)

	appParts.DeployApp(istructs.AppQName_test1_app2, appDef_2_v2, MockEngines(2, 2, 2))
	appParts.DeployApp(istructs.AppQName_test1_app1, appDef_1_v2, MockEngines(2, 2, 2))

	a2_v2_p1, err := appParts.Borrow(istructs.AppQName_test1_app2, 1, appparts.ProcKind_Projector)
	if err != nil {
		panic(err)
	}
	defer a2_v2_p1.Release()

	report(a2_v2_p1)

	a1_v2_p1, err := appParts.Borrow(istructs.AppQName_test1_app1, 1, appparts.ProcKind_Command)
	if err != nil {
		panic(err)
	}
	defer a2_v2_p1.Release()

	report(a1_v2_p1)

	// Output:
	// *** Add ver 1 ***
	// test1/app1 partition 1
	// - CDoc «ver.info» app-1 ver.1
	// test1/app2 partition 1
	// - CDoc «ver.info» app-2 ver.1
	// *** Update to ver 2 ***
	// test1/app2 partition 1
	// - CDoc «ver.info» app-2 ver.2
	// test1/app1 partition 1
	// - CDoc «ver.info» app-1 ver.2
}
