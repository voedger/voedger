/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package apppartsctl

import (
	"github.com/voedger/voedger/pkg/iservices"
)

// IAppPartitionsController is a service that creates, updates (replaces) and deletes applications partitions.
type IAppPartitionsController interface {
	iservices.IService
}
