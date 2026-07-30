/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package iauthnzimpl

import (
	"github.com/voedger/voedger/pkg/iauthnz"
	"github.com/voedger/voedger/pkg/istructs"
)

func NewDefaultAuthenticator(subjectRolesGetter SubjectGetterFunc, isDeviceAllowedFuncs IsDeviceAllowedFuncs) iauthnz.IAuthenticator {
	return &implIAuthenticator{
		subjectRolesGetter:   subjectRolesGetter,
		isDeviceAllowedFuncs: isDeviceAllowedFuncs,
	}
}

var TestIsDeviceAllowedFuncs = IsDeviceAllowedFuncs{
	istructs.AppQName_test1_app1: func(istructs.IAppStructs, istructs.WSID, istructs.WSID) (ok bool, err error) {
		return true, nil
	},
}
