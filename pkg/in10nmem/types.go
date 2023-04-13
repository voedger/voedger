/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * Aleksei Ponomarev
 */

package in10nmem

import (
	"github.com/untillpro/voedger/pkg/in10n"
	istructs "github.com/untillpro/voedger/pkg/istructs"
)

type UpdateUnit struct {
	Projection in10n.ProjectionKey
	Offset     istructs.Offset
}
