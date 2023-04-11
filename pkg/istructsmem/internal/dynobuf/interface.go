/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package dynobuf

import (
	"github.com/untillpro/dynobuffers"
	"github.com/untillpro/voedger/pkg/schemas"
)

// Map of dynobuffer schemas by schema name
type DynoBufSchemasCache map[schemas.QName]*dynobuffers.Scheme
