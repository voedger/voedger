/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package apppartsctl

import (
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/cluster"
)

// Returns a new instance of IAppPartitionsController.
func New(parts appparts.IAppPartitions, apps []cluster.BuiltInApp) (ctl IAppPartitionsController, cleanup func(), err error) {
	return newAppPartitionsController(parts, apps)
}
