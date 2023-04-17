/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package iauthnz

import "github.com/voedger/voedger/pkg/istructs"

// Proposed NewDefaultAuthorizer() signature
// One per HVM
type NewDefaultAuthorizerType func() IAuthorizer

type IAuthorizer interface {
	Authorize(app istructs.IAppStructs, principals []Principal, req AuthzRequest) (ok bool, err error)
}
