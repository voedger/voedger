/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package signupin

import (
	"context"
	"fmt"

	"github.com/untillpro/airs-bp3/utils"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	istructsmem "github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/itokens"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
)

func ProvideQryRefreshPrincipalToken(cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder, itokens itokens.ITokens) {
	cfg.Resources.Add(istructsmem.NewQueryFunction(
		appdef.NewQName(appdef.SysPackage, "RefreshPrincipalToken"),
		appDefBuilder.AddStruct(appdef.NewQName(appdef.SysPackage, "RefreshPrincipalTokenParams"), appdef.DefKind_Object).
			AddField(field_ExistingPrincipalToken, appdef.DataKind_string, true).QName(),
		appDefBuilder.AddStruct(appdef.NewQName(appdef.SysPackage, "RefreshPrincipalTokenResult"), appdef.DefKind_Object).
			AddField(field_NewPrincipalToken, appdef.DataKind_string, true).QName(),
		provideRefreshPrincipalTokenExec(itokens),
	))
}

func provideRefreshPrincipalTokenExec(itokens itokens.ITokens) istructsmem.ExecQueryClosure {
	return func(_ context.Context, _ istructs.IQueryFunction, args istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) (err error) {
		existingPrincipalToken := args.ArgumentObject.AsString(field_ExistingPrincipalToken)

		principalPayload := payloads.PrincipalPayload{}
		gp, err := utils.GetPayloadRegistry(itokens, existingPrincipalToken, &principalPayload)
		if err != nil {
			return err
		}

		newPrincipalToken, err := itokens.IssueToken(gp.AppQName, gp.Duration, &principalPayload)
		if err != nil {
			return fmt.Errorf("token issue failed: %w", err)
		}

		issuePrincipalTokenRR := &issuePrincipalTokenRR{
			principalToken: newPrincipalToken,
		}
		return callback(issuePrincipalTokenRR)
	}
}

type issuePrincipalTokenRR struct {
	istructs.NullObject
	principalToken string
}

func (ipt *issuePrincipalTokenRR) AsString(string) string {
	return ipt.principalToken
}
