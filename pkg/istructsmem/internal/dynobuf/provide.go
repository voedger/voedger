/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package dynobuf

// Returns new dynobuffer schemas cache
func NewSchemasCache() DynoBufSchemasCache {
	return newSchemasCache()
}
