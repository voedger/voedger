/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package vers

// Creates new versions storage for application structures.
func New() *Versions {
	return newVersions()
}
