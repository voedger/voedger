/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Alisher Nurmanov
 */

package ihttp

import (
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
)

func NewIRouterStorage(appStorageProvider istorage.IAppStorageProvider) (IRouterStorage, error) {
	return appStorageProvider.AppStorage(istructs.AppQName_sys_router)
}
