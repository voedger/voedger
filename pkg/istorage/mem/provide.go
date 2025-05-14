/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package mem

import (
	"github.com/voedger/voedger/pkg/goutils/timeu"
	"github.com/voedger/voedger/pkg/istorage"
)

func Provide(iTime timeu.ITime) istorage.IAppStorageFactory {
	return &appStorageFactory{
		storages: map[string]*storageWithLock{},
		iTime:    iTime,
	}
}
