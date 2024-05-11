/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Alisher Nurmanov
 */

package ihttp

import (
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
)

func NewIRouterStorage(appStorageInitializer istorage.IAppStorageInitializer) (IRouterStorage, error) {
	if err := appStorageInitializer.Init(istructs.AppQName_sys_router); err != nil {
		return nil, err
	}
	return appStorageInitializer.AppStorage(istructs.AppQName_sys_router)
}
