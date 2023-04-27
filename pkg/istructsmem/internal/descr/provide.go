/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

func Provide(app istructs.IAppStructs, rateLimits map[appdef.QName]map[istructs.RateLimitKind]istructs.RateLimit, uniques map[appdef.QName][][]string) *Application {
	a := newApplication()
	a.read(app, rateLimits, uniques)
	return a
}
