/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package dynobuf

// Creates and returns new dynobuffer schemes
func New() *DynoBufSchemes {
	return newSchemes()
}
