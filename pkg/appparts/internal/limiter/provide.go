/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package limiter

import "github.com/voedger/voedger/pkg/appdef"

func New(app appdef.IAppDef) *Limiter {
	return &Limiter{app: app}
}
