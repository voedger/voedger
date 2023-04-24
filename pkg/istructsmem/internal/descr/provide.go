/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import (
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/schemas"
)

func Provide(app istructs.IAppStructs, rateLimits map[schemas.QName]map[istructs.RateLimitKind]istructs.RateLimit, uniques map[schemas.QName][][]string) *Application {
	a := newApplication()
	a.read(app, rateLimits, uniques)
	return a
}
