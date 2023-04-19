/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package containers

// Identificator for Container name
type ContainerID uint16

// QNames system view
type Containers struct {
	containers map[string]ContainerID
	ids        map[ContainerID]string
	lastID     ContainerID
	changes    uint
}
