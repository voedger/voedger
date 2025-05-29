/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Michael Saigachenko
 */
package query2

import (
	"context"
	"fmt"
	"net/http"

	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/coreutils"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
)

// [~server.authnz/cmp.authRefreshHandler~impl]
func authRefreshHandler() apiPathHandler {
	return apiPathHandler{
		exec: func(ctx context.Context, qw *queryWork) (err error) {
			if qw.msg.Token() == "" {
				return coreutils.NewHTTPErrorf(http.StatusUnauthorized, fmt.Errorf("authorization header is empty"))
			}

			url := fmt.Sprintf("api/v2/apps/%s/%s/workspaces/%d/queries/sys.RefreshPrincipalToken", qw.msg.AppQName().Owner(), qw.msg.AppQName().Name(),
				qw.principalPayload.ProfileWSID)
			resp, err := qw.federation.QueryNoRetry(url, coreutils.WithAuthorizeBy(qw.msg.Token()))
			if err != nil {
				return err
			}

			if resp.IsEmpty() {
				return coreutils.NewHTTPErrorf(http.StatusInternalServerError, "sys.RefreshPrincipalToken response is empty")
			}

			newToken := resp.QPv2Response.Result()["NewPrincipalToken"].(string)
			if len(newToken) == 0 {
				return coreutils.NewHTTPErrorf(http.StatusInternalServerError, "sys.RefreshPrincipalToken response contains empty token")
			}

			payload := payloads.PrincipalPayload{}
			gp, err := qw.appStructs.AppTokens().ValidateToken(newToken, &payload)
			if err != nil {
				return err
			}
			expiresIn := gp.Duration.Seconds()
			json := fmt.Sprintf(`{"%s": "%s", "%s": %d, "%s": %d}`, fieldPrincipalToken, newToken, fieldExpiresIn, int(expiresIn), fieldWSID, qw.principalPayload.ProfileWSID)
			return qw.msg.Responder().Respond(bus.ResponseMeta{ContentType: coreutils.ContentType_ApplicationJSON, StatusCode: http.StatusOK}, json)

		},
	}
}
