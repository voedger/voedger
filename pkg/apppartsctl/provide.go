/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package apppartsctl

import (
	"github.com/voedger/voedger/pkg/appparts"
)

// Returns a new instance of IAppPartitionsController.
func New(parts appparts.IAppPartitions) (ctl IAppPartitionsController, cleanup func(), err error) {
	return newAppPartitionsController(parts)
}
