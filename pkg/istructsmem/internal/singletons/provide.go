/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package singletons

// Creates and returns new sigletons system view
func New() *Singletons {
	return newSingletons()
}
