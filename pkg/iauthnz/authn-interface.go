/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package iauthnz

import (
	"context"

	"github.com/voedger/voedger/pkg/istructs"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
)

// Proposed NewDefaultAuthenticator() signature
// One per VVM
type NewDefaultAuthenticatorType func() IAuthenticator

type IAuthenticator interface {
	// if err == nil then len(principals) > 0
	// principals[0] is author - put to event? like to show who is the author of the event?
	Authenticate(requestContext context.Context, app istructs.IAppStructs, appTokens istructs.IAppTokens, req AuthnRequest) (principals []Principal, payload payloads.PrincipalPayload, err error)
}
