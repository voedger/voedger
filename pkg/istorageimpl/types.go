/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package istorageimpl

import (
	"sync"

	"github.com/untillpro/voedger/pkg/istorage"
	"github.com/untillpro/voedger/pkg/istructs"
)

type implIAppStorageProvider struct {
	cache map[istructs.AppQName]istorage.IAppStorage
	asf   istorage.IAppStorageFactory
	lock  sync.Mutex
	//nolint
	metaStorage istorage.IAppStorage
	suffix      string // used in tests only
}
