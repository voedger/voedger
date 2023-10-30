/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package apppartsctl

import "github.com/voedger/voedger/pkg/appparts"

func New(api appparts.IAppPartitionsAPI) (ctl IAppPartitionsController, cleanup func(), err error) {
	return newAppPartitionsController(api)
}
