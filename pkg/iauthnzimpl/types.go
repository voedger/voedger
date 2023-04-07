/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package iauthnzimpl

import (
	"context"

	"github.com/untillpro/voedger/pkg/iauthnz"
	"github.com/untillpro/voedger/pkg/istructs"
)

type implIAuthorizer struct {
	acl ACL
}

type implIAuthenticator struct {
	subjectRolesGetter SubjectGetterFunc
}

type ACElem struct {
	desc    string
	pattern PatternType
	policy  ACPolicyType
}

type ACL []ACElem

type PatternType struct {
	opKindsPattern    []iauthnz.OperationKindType
	principalsPattern [][]iauthnz.Principal // first OR, second AND
	qNamesPattern     []istructs.QName
}

type ACPolicyType int

type SubjectGetterFunc = func(requestContext context.Context, name string, as istructs.IAppStructs, wsid istructs.WSID) ([]istructs.QName, error)
