/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package provider

import (
	"sync"

	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
)

type implIAppStorageProvider struct {
	cache map[istructs.AppQName]istorage.IAppStorage
	asf   istorage.IAppStorageFactory
	lock  sync.Mutex
	//nolint
	metaStorage istorage.IAppStorage
	suffix      string // used in tests only
}
