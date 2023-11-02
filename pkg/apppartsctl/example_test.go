/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package apppartsctl_test

import (
	"context"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/apppartsctl"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istorageimpl"
	"github.com/voedger/voedger/pkg/istructs"
)

func Example() {
	storage := istorageimpl.Provide(istorage.ProvideMem(), "")
	appDef := func(comment ...string) appdef.IAppDef {
		adb := appdef.New()
		adb.AddCDoc(appdef.NewQName("test", "doc")).SetComment(comment...)
		app, err := adb.Build()
		if err != nil {
			panic(err)
		}
		return app
	}

	ctl, cleanup, err := apppartsctl.New(storage, []apppartsctl.BuiltInApp{
		{Name: istructs.AppQName_test1_app1,
			Def:      appDef("first app", "seven partitions"),
			NumParts: 7},
		{Name: istructs.AppQName_test1_app2,
			Def:      appDef("second app", "ten partitions"),
			NumParts: 10},
	})

	if err != nil {
		panic(err)
	}
	defer cleanup()

	err = ctl.Prepare()
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	ctl.Run(ctx)

	cancel()
}
