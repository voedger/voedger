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
		checkRateLimit:   authLoginCheckRateLimit,
		setRequestType:   authLoginSetRequestType,
		exec:             authLoginExec,
	}
}

func authLoginCheckRateLimit(ctx context.Context, qw *queryWork) error {
	if qw.appStructs.IsFunctionRateLimitsExceeded(qw.msg.QName(), qw.msg.WSID()) {
		return coreutils.NewSysError(http.StatusTooManyRequests)
	}
	return nil
}

func authLoginSetRequestType(ctx context.Context, qw *queryWork) error {
	switch qw.iWorkspace {
	case nil:
		// workspace is dummy
		if qw.iQuery = appdef.Query(qw.appStructs.AppDef().Type, qw.msg.QName()); qw.iQuery == nil {
			return coreutils.NewHTTPErrorf(http.StatusBadRequest, fmt.Sprintf("query %s does not exist", qw.msg.QName()))
		}
	default:
		if qw.iQuery = appdef.Query(qw.iWorkspace.Type, qw.msg.QName()); qw.iQuery == nil {
			return coreutils.NewHTTPErrorf(http.StatusBadRequest, fmt.Sprintf("query %s does not exist in %v", qw.msg.QName(), qw.iWorkspace))
		}
	}
	return nil
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
