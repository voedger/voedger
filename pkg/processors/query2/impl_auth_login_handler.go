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
	"github.com/voedger/voedger/pkg/sys/authnz"
)

// [~server.apiv2.auth/cmp.authLoginHandler~impl]
func authLoginHandler() apiPathHandler {
	return apiPathHandler{
		requestOpKind:    appdef.OperationKind_Execute,
		handlesQueryArgs: true,
		checkRateLimit:   queryRateLimitExceeded,
		setRequestType:   querySetRequestType,
		exec:             authLoginExec,
	}
}
func authLoginExec(ctx context.Context, qw *queryWork) (err error) {
	var principalTokenObj istructs.IObject
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
		return
	}
	if principalTokenObj == nil {
		return coreutils.NewHTTPErrorf(http.StatusInternalServerError, "principal token object is nil")
	}
	principalToken := principalTokenObj.AsString("PrincipalToken")
	wsid := principalTokenObj.AsInt64("WSID")
	expiresIn := authnz.DefaultPrincipalTokenExpiration.Seconds()
	json := fmt.Sprintf(`{
  	"PrincipalToken": "%s",
  	"ExpiresIn": %d,
  	"WSID": %d
	}`, principalToken, int(expiresIn), wsid)
	return qw.msg.Responder().Respond(bus.ResponseMeta{ContentType: coreutils.ContentType_ApplicationJSON, StatusCode: http.StatusOK}, json)
}
