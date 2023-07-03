/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package signupin

import (
	"context"

	"github.com/voedger/voedger/pkg/iauthnz"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"golang.org/x/exp/slices"
)

type enrichPrincipalTokenRR struct {
	istructs.NullObject
	enrichedToken string
}

func (r *enrichPrincipalTokenRR) AsString(string) string {
	return r.enrichedToken
}

// targetApp/parentWS/q.sys.EnrichPrincipalToken
// basic auth, WorkspaceOwner
func provideExecQryEnrichPrincipalToken(atf payloads.IAppTokensFactory) istructsmem.ExecQueryClosure {
	return func(ctx context.Context, qf istructs.IQueryFunction, args istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) (err error) {
		principalPayload := args.Workpiece.(interface {
			GetPrincipalPayload() payloads.PrincipalPayload
		}).GetPrincipalPayload()

		appQName := args.Workpiece.(interface{ AppQName() istructs.AppQName }).AppQName()
		appTokens := atf.New(appQName)

		principals := args.Workpiece.(interface{ GetPrincipals() []iauthnz.Principal }).GetPrincipals()
		for _, prn := range principals {
			if prn.Kind != iauthnz.PrincipalKind_Role {
				continue
			}
			newRole := payloads.RoleType{
				WSID:  prn.WSID,
				QName: prn.QName,
			}
			if !slices.Contains(principalPayload.Roles, newRole) {
				principalPayload.Roles = append(principalPayload.Roles, newRole)
			}
		}

		enrichedToken, err := appTokens.IssueToken(DefaultPrincipalTokenExpiration, &principalPayload)
		if err != nil {
			return err
		}
		return callback(&enrichPrincipalTokenRR{enrichedToken: enrichedToken})
	}
}
