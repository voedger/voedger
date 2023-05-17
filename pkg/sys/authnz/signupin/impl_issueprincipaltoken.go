/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package signupin

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/voedger/voedger/pkg/istructs"
	istructsmem "github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/itokens"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/sys/authnz"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

// q.sys.IssuePrincipalToken
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

func provideIssuePrincipalTokenExec(asp istructs.IAppStructsProvider, itokens itokens.ITokens) istructsmem.ExecQueryClosure {
	return func(ctx context.Context, qf istructs.IQueryFunction, args istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) (err error) {
		login := args.ArgumentObject.AsString(authnz.Field_Login)
		appName := args.ArgumentObject.AsString(Field_AppName)

		appQName, err := istructs.ParseAppQName(appName)
		if err != nil {
			// notest
			// validated already on c.sys.CreateLogin
			return err
		}
		if err != nil {
			return coreutils.NewHTTPErrorf(http.StatusBadRequest, "failed to parse app qualified name", appQName.String(), ":", err)
		}

		as, err := asp.AppStructs(appQName)
		if err != nil {
			if errors.Is(err, istructs.ErrAppNotFound) {
				return coreutils.NewHTTPErrorf(http.StatusBadRequest, "unknown application ", appName)
			}
			return err
		}

		if err = CheckAppWSID(login, args.Workspace, as.WSAmount()); err != nil {
			return err
		}

		cdocLogin, err := GetCDocLogin(login, args.State, args.Workspace, appName)
		if err != nil {
			return err
		}

		if err = CheckPassword(cdocLogin, args.ArgumentObject.AsString(field_Passwrd)); err != nil {
			return err
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
		if result.principalToken, err = itokens.IssueToken(appQName, DefaultPrincipalTokenExpiration, &principalPayload); err != nil {
			return fmt.Errorf("principal token issue failed: %w", err)
		}

		return callback(result)
	}
}
