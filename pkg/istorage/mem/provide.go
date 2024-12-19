/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package mem

import (
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istorage"
)

func Provide(iTime coreutils.ITime, iSleeper coreutils.Sleeper) istorage.IAppStorageFactory {
	return &appStorageFactory{
		storages: map[string]map[string]map[string]dataWithTTL{},
		iTime:    iTime,
		iSleeper: iSleeper,
	}
}
