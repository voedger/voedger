/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package provider

import (
	"github.com/google/uuid"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istorage"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

// keyspaceNameSuffix is used in tests only
// see https://dev.untill.com/projects/#!638565
func Provide(asf istorage.IAppStorageFactory, keyspaceNameSuffix ...string) istorage.IAppStorageProvider {
	res := &implIAppStorageProvider{
		asf:   asf,
		cache: map[appdef.AppQName]istorage.IAppStorage{},
	}
	if coreutils.IsTest() {
		if len(keyspaceNameSuffix) > 0 && len(keyspaceNameSuffix[0]) > 0 {
			res.suffix = keyspaceNameSuffix[0]
		} else {
			res.suffix = uuid.NewString()
		}
	}
	return res
}
