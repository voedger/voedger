/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package vers

// Creates and returns new versions storage
func New() *Versions {
	return newVersions()
}
