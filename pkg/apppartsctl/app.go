/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package apppartsctl

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

type application struct {
	name   istructs.AppQName
	appDef appdef.IAppDef
	parts  func(func(istructs.PartitionID))
}

func newApplication(name istructs.AppQName, appDef appdef.IAppDef, parts func(func(istructs.PartitionID))) *application {
	return &application{name: name, appDef: appDef, parts: parts}
}

func (a application) AppName() istructs.AppQName { return a.name }

func (a application) AppDef() appdef.IAppDef { return a.appDef }

func (a application) Partitions(parts func(istructs.PartitionID)) { a.parts(parts) }
