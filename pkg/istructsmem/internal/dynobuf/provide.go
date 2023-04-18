/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package dynobuf

import "github.com/voedger/voedger/pkg/schemas"

// Returns new dynobuffer schemas cache
func NewSchemasCache(schemas *schemas.SchemasCache) DynoBufSchemasCache {
	return newSchemasCache(schemas)
}
