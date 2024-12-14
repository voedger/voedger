/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package limiter

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/irates"
	"github.com/voedger/voedger/pkg/istructs"
)

type Limiter struct {
	app     appdef.IAppDef
	buckets irates.IBuckets
}

// Return is specified resource (command, query or structure) usage limit is exceeded.
//
// If resource usage is exceeded then returns name of first exceeded limit.
func (l *Limiter) Exceeded(resource appdef.QName, operation appdef.OperationKind, workspace istructs.WSID, remoteAddr string) (bool, appdef.QName) {
	return false, appdef.NullQName
}
