/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import (
	"github.com/untillpro/voedger/pkg/istructs"
)

func Provide(app istructs.IAppStructs, rateLimits map[istructs.QName]map[istructs.RateLimitKind]istructs.RateLimit, uniques map[istructs.QName][][]string) *Application {
	a := newApplication()
	a.read(app, rateLimits, uniques)
	return a
}
