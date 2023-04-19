/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package containers

import (
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/schemas"
)

// Identificator for Container name
type ContainerNameID uint16

// QNames system view
type Containers struct {
	storage     istorage.IAppStorage
	constainers map[schemas.QName]ContainerNameID
	ids         map[ContainerNameID]schemas.QName
	lastID      ContainerNameID
	changes     uint
}
