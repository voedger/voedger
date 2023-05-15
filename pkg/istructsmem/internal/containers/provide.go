/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package containers

// Constructs and return new containers system view
func New() *Containers {
	return newContainers()
}
