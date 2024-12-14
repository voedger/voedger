/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package limiter

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/irates"
)

func New(app appdef.IAppDef, buckets irates.IBuckets) *Limiter {
	return &Limiter{app: app, buckets: buckets}
}
