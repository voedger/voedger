/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package containers

// Identifier for Container name
type ContainerID uint16

// Container IDs system view
//
//	Use GetID() to obtain container ID by its name.
//	Use GetContainer() to obtain container name by its ID.
//	Use Prepare() to load container IDs from storage.
type Containers struct {
	containers map[string]ContainerID
	ids        map[ContainerID]string
	lastID     ContainerID
	changes    uint
}
