/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import (
	"github.com/voedger/voedger/pkg/appdef"
)

func Provide(name appdef.AppQName, app appdef.IAppDef) *Application {
	a := newApplication()
	a.read(name, app)
	return a
}
