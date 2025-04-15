/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Michael Saigachenko
 */
package query2

import (
	"context"
	"encoding/json"
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

			login := qw.msg.QueryParams().Argument["Login"].(string)
			if login == "" {
				return coreutils.NewHTTPErrorf(http.StatusBadRequest, fmt.Errorf("login is empty"))
			}
			password := qw.msg.QueryParams().Argument["Password"].(string)

			pseudoWSID := coreutils.GetPseudoWSID(istructs.NullWSID, login, istructs.CurrentClusterID())

			url := fmt.Sprintf(`api/v2/users/%s/apps/%s/workspaces/%d/queries/registry.IssuePrincipalToken?arg=%s`,
				istructs.SysOwner, istructs.AppQName_sys_registry.Name(), pseudoWSID,
				url.QueryEscape(fmt.Sprintf(`{"Login":"%s", "Password":"%s", "AppName": "%s"}`, login, password, qw.msg.AppQName().Name())))
			resp, err := qw.federation.Query(url)
			if err != nil {
				return err
			}

			var jsonData map[string]interface{}
			if err = json.Unmarshal([]byte(resp.Body), &jsonData); err != nil {
				return fmt.Errorf("sys.IssuePrincipalToken response is not JSON: %s", err.Error())
			}
			if len(jsonData) == 0 || len(jsonData["results"].([]interface{})) == 0 {
				return errors.New("sys.IssuePrincipalToken response is empty")
			}
			if len(jsonData["results"].([]interface{})) > 1 {
				return errors.New("sys.IssuePrincipalToken response contains more than one result")
			}
			token := jsonData["results"].([]interface{})[0].(map[string]interface{})["PrincipalToken"].(string)
			wsError := jsonData["results"].([]interface{})[0].(map[string]interface{})["WSError"].(string)
			wsid := jsonData["WSID"].([]interface{})[0].(map[string]interface{})["WSID"].(float64)

			if wsError != "" {
				return coreutils.NewHTTPErrorf(http.StatusUnauthorized, fmt.Errorf("login error: %s", wsError))
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
