/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package payloads

import (
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

func (at *implIAppTokens) IssueToken(duration time.Duration, pointerToPayload interface{}) (token string, err error) {
	return at.itokens.IssueToken(at.appQName, duration, pointerToPayload)
}

func (at *implIAppTokens) ValidateToken(token string, pointerToPayload interface{}) (gp istructs.GenericPayload, err error) {
	if gp, err = at.itokens.ValidateToken(token, pointerToPayload); err != nil {
		return
	}
	if gp.AppQName != at.appQName {
		err = ErrTokenIssuedForAnotherApp
	}
	return
}

func (atf *implIAppTokensFactory) New(app appdef.AppQName) istructs.IAppTokens {
	return &implIAppTokens{
		itokens:  atf.tokens,
		appQName: app,
	}
}
