/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Creates and return new application definition builder
func New() IAppDefBuilder {
	return newAppDef()
}
