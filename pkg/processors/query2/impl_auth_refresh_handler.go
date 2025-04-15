/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Michael Saigachenko
 */
package query2

import (
	"context"
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
)

// [~server.apiv2.auth/cmp.authRefreshHandler~impl]
func authRefreshHandler() apiPathHandler {
	return apiPathHandler{
		requestOpKind:  appdef.OperationKind_Execute,
		checkRateLimit: queryRateLimitExceeded,
		setRequestType: querySetRequestType,
		exec:           authRefreshExec,
	}
}
func authRefreshClarifyWSID(ctx context.Context, qw *queryWork) (wsid istructs.WSID, err error) {
	// [~server.apiv2.auth/cmp.authRefreshHandler.WSID~impl]
	var principalPayload payloads.PrincipalPayload
	if _, err = qw.appStructs.AppTokens().ValidateToken(qw.msg.Token(), &principalPayload); err != nil {
		return istructs.NullWSID, err
	}
	return principalPayload.ProfileWSID, nil
}
func authRefreshExec(ctx context.Context, qw *queryWork) (err error) {
	var principalTokenObj istructs.IObject
	var gp istructs.GenericPayload
	qw.callbackFunc = func(o istructs.IObject) (err error) {
		principalTokenObj = o
		return nil
	}
	defer func() {
		if r := recover(); r != nil {
			stack := string(debug.Stack())
			err = fmt.Errorf("%v\n%s", r, stack)
		}
	}()
	err = qw.appPart.Invoke(ctx, qw.msg.QName(), qw.state, qw.state)
	if err != nil {
		return err
	}
	if principalTokenObj == nil {
		return coreutils.NewHTTPErrorf(http.StatusInternalServerError, "principal token object is nil")
	}
	newToken := principalTokenObj.AsString("NewPrincipalToken")

	payload := payloads.PrincipalPayload{}
	gp, err = qw.appStructs.AppTokens().ValidateToken(newToken, &payload)
	if err != nil {
		return err
	}
	expiresIn := gp.Duration.Seconds()
	json := fmt.Sprintf(`{
  	"PrincipalToken": "%s",
  	"ExpiresIn": %d,
  	"WSID": %d
	}`, newToken, int(expiresIn), qw.principalPayload.ProfileWSID)
	return qw.msg.Responder().Respond(bus.ResponseMeta{ContentType: coreutils.ContentType_ApplicationJSON, StatusCode: http.StatusOK}, json)
}
