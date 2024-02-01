/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package registry

import (
	"context"
	"fmt"
	"net/http"

	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/itokens"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/sys/authnz"
	coreutils "github.com/voedger/voedger/pkg/utils"
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

		appQName, err := istructs.ParseAppQName(appName)
		if err != nil {
			// notest
			// validated already on c.registry.CreateLogin
			return err
		}
		if err != nil {
			return coreutils.NewHTTPErrorf(http.StatusBadRequest, "failed to parse app qualified name", appQName.String(), ":", err)
		}

		// as, err := asp.AppStructs(appQName)
		// if err != nil {
		// 	if errors.Is(err, istructs.ErrAppNotFound) {
		// 		return coreutils.NewHTTPErrorf(http.StatusBadRequest, "unknown application ", appName)
		// 	}
		// 	return err
		// }

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

		// issue principal token
		principalPayload := payloads.PrincipalPayload{
			Login:       args.ArgumentObject.AsString(authnz.Field_Login),
			SubjectKind: istructs.SubjectKindType(cdocLogin.AsInt32(authnz.Field_SubjectKind)),
			ProfileWSID: istructs.WSID(result.profileWSID),
		}
		if result.principalToken, err = itokens.IssueToken(appQName, authnz.DefaultPrincipalTokenExpiration, &principalPayload); err != nil {
			return fmt.Errorf("principal token issue failed: %w", err)
		}

		return callback(result)
	}
}
