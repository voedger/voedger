/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package apppartsctl

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
)

// Returns a new instance of IAppPartitionsController.
func New(storages istorage.IAppStorageProvider, apps []BuiltInApp) (ctl IAppPartitionsController, cleanup func(), err error) {
	return newAppPartitionsController(storages, apps)
}

type BuiltInApp struct {
	Name     istructs.AppQName
	Def      appdef.IAppDef
	NumParts int
}
