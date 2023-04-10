/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package iauthnzimpl

import (
	"github.com/untillpro/voedger/pkg/iauthnz"
)

func NewDefaultAuthorizer() iauthnz.IAuthorizer {
	return &implIAuthorizer{acl: defaultACL}
}

func NewDefaultAuthenticator(subjectRolesGetter SubjectGetterFunc) iauthnz.IAuthenticator {
	return &implIAuthenticator{subjectRolesGetter: subjectRolesGetter}
}
