/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package authnz

import (
	"context"
	"fmt"

	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/itokens"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/sys/storages"
)

func provideRefreshPrincipalTokenExec(itokens itokens.ITokens) istructsmem.ExecQueryClosure {
	return func(_ context.Context, args istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) (err error) {
		existingPrincipalToken, err := storages.GetPrincipalTokenFromState(args.State)
		if err != nil {
			return err
		}

		principalPayload := payloads.PrincipalPayload{}
		gp, err := payloads.GetPayloadRegistry(itokens, existingPrincipalToken, &principalPayload)
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
