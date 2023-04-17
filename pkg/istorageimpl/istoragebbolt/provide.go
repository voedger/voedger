/*
 * Copyright (c) 2022-present Sigma-Soft, Ltd.
 * @author: Dmitry Molchanovsky
 */

package istoragebbolt

import (
	istorage "github.com/voedger/voedger/pkg/istorage"
)

func Provide(params ParamsType) istorage.IAppStorageFactory {
	return &appStorageFactory{
		bboltParams: params,
	}
}
