/*
 * Copyright (c) 2022-present Sigma-Soft, Ltd.
 * @author: Dmitry Molchanovsky
 */

package bbolt

import (
	"context"

	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istorage"
)

func Provide(ctx context.Context, params ParamsType, iTime coreutils.ITime) istorage.IAppStorageFactory {
	return &appStorageFactory{
		bboltParams: params,
		iTime:       iTime,
		ctx:         ctx,
	}
}
