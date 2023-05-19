/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package containers

import "github.com/voedger/voedger/pkg/istructsmem/internal/vers"

// constants for system container names
const (
	NullContainerID ContainerID = 0 + iota

	ContainerNameIDSysLast ContainerID = 63
)

// maximum Container ID value
const MaxAvailableContainerID = 0xFFFF

// Containers system view versions
const (
	ver01 vers.VersionValue = vers.UnknownVersion + 1

	latestVersion = ver01
)
