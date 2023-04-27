/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package dynobuf

// Returns new dynobuffer schemes
func New() DynoBufSchemes {
	return newSchemes()
}
