/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Michael Saigachenko
 */
package query2

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys/authnz"
)

// [~server.apiv2.auth/cmp.authLoginHandler~impl]
func authLoginHandler() apiPathHandler {
	return apiPathHandler{
		exec: func(ctx context.Context, qw *queryWork) (err error) {

			args := coreutils.MapObject(qw.msg.QueryParams().Argument)
			login, _, err := args.AsString("Login")
			if err != nil {
				return coreutils.NewHTTPError(http.StatusBadRequest, err)
			}
			password, _, err := args.AsString("Password")
			if err != nil {
				return coreutils.NewHTTPError(http.StatusBadRequest, err)
			}

			pseudoWSID := coreutils.GetPseudoWSID(istructs.NullWSID, login, istructs.CurrentClusterID())

			url := fmt.Sprintf(`api/v2/apps/%s/%s/workspaces/%d/queries/registry.IssuePrincipalToken?args=%s`,
				istructs.SysOwner, istructs.AppQName_sys_registry.Name(), pseudoWSID,
				url.QueryEscape(fmt.Sprintf(`{"Login":"%s", "Password":"%s", "AppName": "%s"}`, login, password, qw.msg.AppQName())))
			resp, err := qw.federation.QueryNoRetry(url)
			if err != nil {
				return err
			}
			if resp.IsEmpty() {
				return errors.New("sys.IssuePrincipalToken response is empty")
			}
			token := resp.QPv2Response.Result()["PrincipalToken"].(string)
			wsError := resp.QPv2Response.Result()["WSError"].(string)
			wsid := resp.QPv2Response.Result()["WSID"].(float64)

			if len(wsError) > 0 {
				return errors.New("the login profile is created with an error: " + wsError)
			}

			expiresIn := authnz.DefaultPrincipalTokenExpiration.Seconds()
			json := fmt.Sprintf(`{
				"PrincipalToken": "%s",
				"ExpiresIn": %d,
				"WSID": %d
			}`, token, int(expiresIn), int(wsid))
			return qw.msg.Responder().Respond(bus.ResponseMeta{ContentType: coreutils.ContentType_ApplicationJSON, StatusCode: http.StatusOK}, json)
		},
	}
}
