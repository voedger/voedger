/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package iauthnzimpl

import (
	"github.com/voedger/voedger/pkg/iauthnz"
)

func NewDefaultAuthorizer() iauthnz.IAuthorizer {
	return &implIAuthorizer{acl: defaultACL}
}

func NewDefaultAuthenticator(subjectRolesGetter SubjectGetterFunc, isDeviceAllowedFunc IsDeviceAllowedFunc) iauthnz.IAuthenticator {
	return &implIAuthenticator{
		subjectRolesGetter: subjectRolesGetter,
		isDeviceAllowed:    isDeviceAllowedFunc,
	}
}

func TestIsDeviceAllowedFunc() bool {
	return true
}
