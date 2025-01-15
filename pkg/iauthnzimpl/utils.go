/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package iauthnzimpl

import (
	"context"
	"fmt"
	"strings"
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

func prnsToString(prns []iauthnz.Principal) string {
	if len(prns) == 0 {
		return "<no principals>"
	}
	res := strings.Builder{}
	res.WriteString("[")
	for i := 0; i < len(prns); i++ {
		prn := prns[i]
		switch prn.Kind {
		case iauthnz.PrincipalKind_Host:
			res.WriteString("Host")
		case iauthnz.PrincipalKind_User:
			res.WriteString("User")
		case iauthnz.PrincipalKind_Role:
			res.WriteString("Role")
		case iauthnz.PrincipalKind_Group:
			res.WriteString("Group")
		case iauthnz.PrincipalKind_Device:
			res.WriteString("Device")
		default:
			res.WriteString("<unknown>")
		}
		if prn.QName != appdef.NullQName {
			res.WriteString(" " + prn.QName.String())
		} else {
			res.WriteString(" " + prn.Name)
		}
		if prn.ID > 0 {
			res.WriteString(fmt.Sprintf(",ID %d", prn.ID))
		}
		if prn.WSID > 0 {
			res.WriteString(fmt.Sprintf(",WSID %d", prn.WSID))
		}
		if i != len(prns)-1 {
			res.WriteString(";")
		}
	}
	res.WriteString("]")
	return res.String()
}
