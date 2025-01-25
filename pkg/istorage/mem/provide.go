/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package mem

import (
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istorage"
)

func Provide(iTime coreutils.ITime) istorage.IAppStorageFactory {
	return &appStorageFactory{
		storages: map[string]map[string]map[string]coreutils.DataWithExpiration{},
		iTime:    iTime,
	}
}
