/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package singletons

func New() *Singletons {
	return newSingletons()
}
