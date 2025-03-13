/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package bus

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/goutils/logger"
)

// TODO: CP should send CommandResponse struct itself, not CommandResponse marshaled to a string
func GetCommandResponse(ctx context.Context, requestSender IRequestSender, req Request) (cmdRespMeta ResponseMeta, cmdResp coreutils.CommandResponse, err error) {
	responseCh, responseMeta, responseErr, err := requestSender.SendRequest(ctx, req)
	if err != nil {
		return cmdRespMeta, cmdResp, err
	}
	body := ""
	for elem := range responseCh {
		if len(body) > 0 {
			// notest
			panic(fmt.Sprintf("unexpected response element: %v", elem))
		}
		switch typed := elem.(type) {
		case string:
			body = typed
		case interface{ ToJSON() string }:
			body = typed.ToJSON()
		}
	}
	if *responseErr != nil {
		cmdResp.SysError = coreutils.WrapSysErrorToExact(*responseErr, http.StatusInternalServerError)
		return responseMeta, cmdResp, nil
	}
	if err = json.Unmarshal([]byte(body), &cmdResp); err != nil {
		// notest
		panic(err)
	}
	return responseMeta, cmdResp, nil
}

func ReplyPlainText(responder IResponder, text string) {
	respWriter := responder.BeginCustomResponse(ResponseMeta{ContentType: coreutils.TextPlain, StatusCode: http.StatusOK})
	if err := respWriter.Write(text); err != nil {
		logger.Error(err.Error() + ": failed to send response: " + text)
	}
	respWriter.Close()
}

func ReplyErrf(responder IResponder, status int, args ...interface{}) {
	ReplyErrDef(responder, coreutils.NewHTTPErrorf(status, args...), http.StatusInternalServerError)
}

//nolint:errorlint
func ReplyErrDef(responder IResponder, err error, defaultStatusCode int) {
	res := coreutils.WrapSysErrorToExact(err, defaultStatusCode)
	respWriter := responder.BeginCustomResponse(ResponseMeta{ContentType: coreutils.ApplicationJSON, StatusCode: res.HTTPStatus})
	if err := respWriter.Write(res); err != nil {
		logger.Error(fmt.Sprintf("failed to send error %s: %s", res, err))
	}
}

func ReplyErr(responder IResponder, err error) {
	ReplyErrDef(responder, err, http.StatusInternalServerError)
}

func ReplyJSON(responder IResponder, httpCode int, obj any) {
	respWriter := responder.BeginCustomResponse(ResponseMeta{ContentType: coreutils.ApplicationJSON, StatusCode: httpCode})
	if err := respWriter.Write(obj); err != nil {
		logger.Error(fmt.Sprintf("failed to send %v: %s", obj, err))
	}
}

func ReplyBadRequest(responder IResponder, message string) {
	ReplyErrf(responder, http.StatusBadRequest, message)
}

func replyAccessDenied(responder IResponder, code int, message string) {
	msg := "access denied"
	if len(message) > 0 {
		msg += ": " + message
	}
	ReplyErrf(responder, code, msg)
}

func ReplyAccessDeniedUnauthorized(responder IResponder, message string) {
	replyAccessDenied(responder, http.StatusUnauthorized, message)
}

func ReplyAccessDeniedForbidden(responder IResponder, message string) {
	replyAccessDenied(responder, http.StatusForbidden, message)
}

func ReplyUnauthorized(responder IResponder, message string) {
	ReplyErrf(responder, http.StatusUnauthorized, message)
}

func ReplyInternalServerError(responder IResponder, message string, err error) {
	ReplyErrf(responder, http.StatusInternalServerError, message, ": ", err)
}

func GetTestSendTimeout() SendTimeout {
	if coreutils.IsDebug() {
		return SendTimeout(time.Hour)
	}
	return DefaultSendTimeout
}
