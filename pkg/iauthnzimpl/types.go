/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package iauthnzimpl

import (
	"context"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

type implIAuthenticator struct {
	subjectRolesGetter   SubjectGetterFunc
	isDeviceAllowedFuncs IsDeviceAllowedFuncs
}

type SubjectGetterFunc = func(requestContext context.Context, name string, as istructs.IAppStructs, wsid istructs.WSID) ([]appdef.QName, error)

type IsDeviceAllowedFunc = func(as istructs.IAppStructs, requestWSID istructs.WSID, deviceProfileWSID istructs.WSID) (ok bool, err error)
type IsDeviceAllowedFuncs map[appdef.AppQName]IsDeviceAllowedFunc
