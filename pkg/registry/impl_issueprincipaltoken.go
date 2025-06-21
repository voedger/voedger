/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package registry

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/itokens"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/sys/authnz"
)

// q.registry.IssuePrincipalToken
type iptRR struct {
	istructs.NullObject
	principalToken       string
	profileWSID          int64
	profileCreationError string // like wsError
}

func (q *iptRR) AsInt64(string) int64 { return q.profileWSID }
func (q *iptRR) AsString(name string) string {
	if name == authnz.Field_WSError {
		return q.profileCreationError
	}
	return q.principalToken
}

func provideIssuePrincipalTokenExec(itokens itokens.ITokens) istructsmem.ExecQueryClosure {
	return func(ctx context.Context, args istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) (err error) {
		login := args.ArgumentObject.AsString(authnz.Field_Login)
		appName := args.ArgumentObject.AsString(authnz.Field_AppName)

		appQName, err := appdef.ParseAppQName(appName)
		if err != nil {
			// notest
			// validated already on c.registry.CreateLogin
			return err
		}

		// TODO: check we're called at our AppWSID?

		cdocLogin, doesLoginExist, err := GetCDocLogin(login, args.State, args.WSID, appName)
		if err != nil {
			return err
		}

		if !doesLoginExist {
			return errLoginOrPasswordIsIncorrect
		}

		isPasswordOK, err := CheckPassword(cdocLogin, args.ArgumentObject.AsString(field_Passwrd))
		if err != nil {
			return err
		}

		if !isPasswordOK {
			return errLoginOrPasswordIsIncorrect
		}

		result := &iptRR{
			profileWSID:          cdocLogin.AsInt64(authnz.Field_WSID),
			profileCreationError: cdocLogin.AsString(authnz.Field_WSError),
		}
		if result.profileWSID == 0 || len(result.profileCreationError) > 0 {
			return callback(result)
		}

		// read global globalRoles
		globarRolesStr := cdocLogin.AsString(authnz.Field_GlobalRoles)
		var globalRoles []appdef.QName
		if len(globarRolesStr) > 0 {
			globalRolesStr := strings.Split(cdocLogin.AsString(authnz.Field_GlobalRoles), ",")
			for _, role := range globalRolesStr {
				roleQName, err := appdef.ParseQName(role)
				if err != nil {
					return err
				}
				globalRoles = append(globalRoles, roleQName)
			}
		}

		// issue principal token
		principalPayload := payloads.PrincipalPayload{
			Login:       args.ArgumentObject.AsString(authnz.Field_Login),
			SubjectKind: istructs.SubjectKindType(cdocLogin.AsInt32(authnz.Field_SubjectKind)),
			ProfileWSID: istructs.WSID(result.profileWSID), //nolint G115 since WSID is created by NewWSID()
			GlobalRoles: globalRoles,                       // [~server.authnz.groles/cmp.c.registry.IssuePrincipalToken~impl]
		}
		ttl := time.Duration(args.ArgumentObject.AsInt32(field_TTLHours)) * time.Hour
		if ttl == 0 {
			ttl = authnz.DefaultPrincipalTokenExpiration
		} else if ttl > maxTokenTTLHours*time.Hour {
			return coreutils.NewHTTPErrorf(http.StatusBadRequest, fmt.Errorf("max token TTL hours is %d hours", maxTokenTTLHours))
		}

		if result.principalToken, err = itokens.IssueToken(appQName, ttl, &principalPayload); err != nil {
			return fmt.Errorf("principal token issue failed: %w", err)
		}

		return callback(result)
	}
}
