/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package bus

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
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
		case coreutils.SysError:
			if req.IsAPIV2 {
				body = typed.ToJSON_APIV2()
			} else {
				body = typed.ToJSON_APIV1()
			}
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

func ReadQueryResponse(ctx context.Context, sender IRequestSender, req Request) (resp []map[string]interface{}, err error) {
	respCh, _, respErr, err := sender.SendRequest(ctx, req)
	if err != nil {
		// notest
		return nil, err
	}
	defer func() {
		for range respCh {
		}
	}()
	for elem := range respCh {
		switch typed := elem.(type) {
		case map[string]interface{}:
			resp = append(resp, typed)
		case error:
			return nil, typed
		default:
			return nil, fmt.Errorf("unexpected query result element: %#v", elem)
		}
	}
	return resp, *respErr
}

func ReplyErrf(responder IResponder, status int, args ...interface{}) {
	ReplyErrDef(responder, coreutils.NewHTTPErrorf(status, args...), http.StatusInternalServerError)
}

//nolint:errorlint
func ReplyErrDef(responder IResponder, err error, defaultStatusCode int) {
	res := coreutils.WrapSysErrorToExact(err, defaultStatusCode)
	if err := responder.Respond(ResponseMeta{ContentType: coreutils.ContentType_ApplicationJSON, StatusCode: res.HTTPStatus}, res); err != nil {
		logger.Error(err)
	}
}

func ReplyErr(responder IResponder, err error) {
	ReplyErrDef(responder, err, http.StatusInternalServerError)
}

func ReplyJSON(responder IResponder, httpCode int, obj any) {
	if err := responder.Respond(ResponseMeta{ContentType: coreutils.ContentType_ApplicationJSON, StatusCode: httpCode}, obj); err != nil {
		logger.Error(err)
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

func GetPrincipalToken(request Request) (token string, err error) {
	authHeader := request.Header[coreutils.Authorization]
	if len(authHeader) == 0 {
		return "", nil
	}
	if strings.HasPrefix(authHeader, coreutils.BearerPrefix) {
		return strings.ReplaceAll(authHeader, coreutils.BearerPrefix, ""), nil
	}
	if strings.HasPrefix(authHeader, "Basic ") {
		return getBasicAuthToken(authHeader)
	}
	return "", errors.New("unsupported Authorization header: " + authHeader)
}

func getBasicAuthToken(authHeader string) (token string, err error) {
	headerValue := strings.ReplaceAll(authHeader, "Basic ", "")
	headerValueBytes, err := base64.StdEncoding.DecodeString(headerValue)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 Basic Authorization header value: %w", err)
	}
	headerValue = string(headerValueBytes)
	if strings.Count(headerValue, ":") != 1 {
		return "", errors.New("unexpected Basic Authorization header format")
	}
	return strings.ReplaceAll(headerValue, ":", ""), nil
}
