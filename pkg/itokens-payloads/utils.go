/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package payloads

import (
	"fmt"

	"github.com/voedger/voedger/pkg/istructs"
	itokens "github.com/voedger/voedger/pkg/itokens"
)

func GetSystemPrincipalToken(itokens itokens.ITokens, appQName istructs.AppQName) (string, error) {
	systemPrincipalToken, err := itokens.IssueToken(appQName, DefaultSystemPrincipalDuration, &systemPrincipalPayload)
	if err != nil {
		return "", fmt.Errorf("failed to issue system principal token: %w", err)
	}
	return systemPrincipalToken, nil
}

func GetSystemPrincipalTokenApp(appTokens istructs.IAppTokens) (string, error) {
	systemPrincipalToken, err := appTokens.IssueToken(DefaultSystemPrincipalDuration, &systemPrincipalPayload)
	if err != nil {
		return "", fmt.Errorf("failed to issue system principal token: %w", err)
	}
	return systemPrincipalToken, nil
}

func IsSystemPrincipal(principalPayload *PrincipalPayload) bool {
	return principalPayload != nil && principalPayload.ProfileWSID == istructs.NullWSID
}
