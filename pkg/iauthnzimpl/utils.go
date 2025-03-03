/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package iauthnzimpl

import (
	"context"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iauthnz"
	"github.com/voedger/voedger/pkg/istructs"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
)

var TestSubjectRolesGetter = func(context.Context, string, istructs.IAppStructs, istructs.WSID) ([]appdef.QName, error) {
	return nil, nil
}

func IssueAPIToken(appTokens istructs.IAppTokens, duration time.Duration, roles []appdef.QName, wsid istructs.WSID, currentPrincipalPayload payloads.PrincipalPayload) (token string, err error) {
	if wsid == istructs.NullWSID {
		return "", ErrPersonalAccessTokenOnNullWSID
	}
	for _, roleQName := range roles {
		if iauthnz.IsSystemRole(roleQName) {
			return "", ErrPersonalAccessTokenOnSystemRole
		}
		currentPrincipalPayload.Roles = append(currentPrincipalPayload.Roles, payloads.RoleType{
			WSID:  wsid,
			QName: roleQName,
		})
	}
	currentPrincipalPayload.IsAPIToken = true
	return appTokens.IssueToken(duration, &currentPrincipalPayload)
}
