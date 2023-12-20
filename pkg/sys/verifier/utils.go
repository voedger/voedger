/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package verifier

import (
	"net/http"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	itokens "github.com/voedger/voedger/pkg/itokens"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func NewVerificationToken(entity string, field, value string, kind appdef.VerificationKind, targetWSID istructs.WSID, itokens itokens.ITokens, appTokens istructs.IAppTokens) (token, code string, err error) {
	verificationCode, err := coreutils.EmailVerificationCode()
	if err != nil {
		// notest
		return "", "", err
	}
	verificationCodeHash := itokens.CryptoHash256([]byte(verificationCode))

	entityQName, err := appdef.ParseQName(entity)
	if err != nil {
		return "", "", err
	}
	vp := payloads.VerificationPayload{
		VerifiedValuePayload: payloads.VerifiedValuePayload{
			VerificationKind: kind,
			Entity:           entityQName,
			Field:            field,
			Value:            value,
			WSID:             targetWSID,
		},
		Hash256: verificationCodeHash,
	}

	token, err = appTokens.IssueToken(VerificationTokenDuration, &vp)
	return token, string(verificationCode), err
}

func IssueVerfiedValueToken(token, code string, appTokens istructs.IAppTokens, itokens itokens.ITokens) (verifiedValueToken string, err error) {
	vp := payloads.VerificationPayload{}
	if _, err = appTokens.ValidateToken(token, &vp); err != nil {
		return "", coreutils.NewHTTPError(http.StatusBadRequest, err)
	}
	if vp.Hash256 != itokens.CryptoHash256([]byte(code)) {
		return "", coreutils.NewHTTPErrorf(http.StatusBadRequest, "invalid verification code")
	}
	if verifiedValueToken, err = appTokens.IssueToken(VerifiedValueTokenDuration, &vp.VerifiedValuePayload); err != nil {
		return "", err
	}
	return
}
