/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package bus

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
)

func TestReplyError(t *testing.T) {
	require := require.New(t)

	type expected struct {
		code  int
		error coreutils.SysError
	}

	testSysError := coreutils.SysError{
		HTTPStatus: http.StatusAlreadyReported,
		Message:    "test error",
		Data:       "dddfd",
		QName:      appdef.NewQName("my", "qname"),
	}

	t.Run("reply errors", func(t *testing.T) {
		err := errors.New("test error")
		cases := []struct {
			desc     string
			f        func(responder IResponder)
			expected expected
		}{
			{
				desc: "ReplyErr",
				f:    func(responder IResponder) { ReplyErr(responder, err) },
				expected: expected{
					code:  http.StatusInternalServerError,
					error: coreutils.SysError{HTTPStatus: http.StatusInternalServerError, Message: err.Error()},
				},
			},
			{
				desc: "ReplyErrf",
				f:    func(responder IResponder) { ReplyErrf(responder, http.StatusAccepted, "test ", "message") },
				expected: expected{
					code:  http.StatusAccepted,
					error: coreutils.SysError{HTTPStatus: http.StatusAccepted, Message: "test message"},
				},
			},
			{
				desc: "ReplyErrorDef",
				f:    func(responder IResponder) { ReplyErrDef(responder, err, http.StatusAccepted) },
				expected: expected{
					code:  http.StatusAccepted,
					error: coreutils.SysError{HTTPStatus: http.StatusAccepted, Message: err.Error()},
				},
			},
			{
				desc: "SysError",
				f: func(responder IResponder) {
					ReplyErrDef(responder, testSysError, http.StatusAccepted)
				},
				expected: expected{
					code:  http.StatusAlreadyReported,
					error: testSysError,
				},
			},
			{
				desc: "ReplyInternalServerError",
				f: func(responder IResponder) {
					ReplyInternalServerError(responder, "test", err)
				},
				expected: expected{
					code:  http.StatusInternalServerError,
					error: coreutils.SysError{HTTPStatus: http.StatusInternalServerError, Message: "test: test error"},
				},
			},
		}

		for _, c := range cases {
			t.Run(c.desc, func(t *testing.T) {
				requestSender := NewIRequestSender(coreutils.MockTime, GetTestSendTimeout(), func(requestCtx context.Context, request Request, responder IResponder) {
					c.f(responder)
				})
				cmdRespMeta, cmdResp, err := GetCommandResponse(context.Background(), requestSender, Request{})
				require.NoError(err)
				require.Equal(coreutils.ApplicationJSON, cmdRespMeta.ContentType)
				require.Equal(c.expected.code, cmdRespMeta.StatusCode)
				require.Equal(c.expected.error, cmdResp.SysError)
			})
		}

	})

	t.Run("http status helpers", func(t *testing.T) {
		cases := []struct {
			statusCode      int
			f               func(responder IResponder, message string)
			expectedMessage string
		}{
			{f: ReplyUnauthorized, statusCode: http.StatusUnauthorized},
			{f: ReplyBadRequest, statusCode: http.StatusBadRequest},
			{f: ReplyAccessDeniedForbidden, statusCode: http.StatusForbidden, expectedMessage: "access denied: test message"},
			{f: ReplyAccessDeniedUnauthorized, statusCode: http.StatusUnauthorized, expectedMessage: "access denied: test message"},
		}

		for _, c := range cases {
			name := runtime.FuncForPC(reflect.ValueOf(c.f).Pointer()).Name()
			name = name[strings.LastIndex(name, ".")+1:]
			t.Run(name, func(t *testing.T) {
				requestSender := NewIRequestSender(coreutils.MockTime, GetTestSendTimeout(), func(requestCtx context.Context, request Request, responder IResponder) {
					go c.f(responder, "test message")
				})
				expectedMessage := "test message"
				if len(c.expectedMessage) > 0 {
					expectedMessage = c.expectedMessage
				}
				meta, resp, err := GetCommandResponse(context.Background(), requestSender, Request{})
				require.NoError(err)
				require.Equal(coreutils.ApplicationJSON, meta.ContentType)
				require.Equal(c.statusCode, resp.SysError.HTTPStatus)
				require.Equal(expectedMessage, resp.SysError.Message)
			})
		}
	})

	t.Run("reply json", func(t *testing.T) {
		testObj := struct {
			Fld1 int
			Fld2 string
		}{Fld1: 42, Fld2: "str"}
		requestSender := NewIRequestSender(coreutils.MockTime, GetTestSendTimeout(), func(requestCtx context.Context, request Request, responder IResponder) {
			ReplyJSON(responder, http.StatusOK, testObj)
		})
		responseCh, responseMeta, responseErr, err := requestSender.SendRequest(context.Background(), Request{})
		require.NoError(err)
		counter := 0
		for elem := range responseCh {
			require.Zero(counter)
			require.Equal(http.StatusOK, responseMeta.StatusCode)
			require.Equal(coreutils.ApplicationJSON, responseMeta.ContentType)
			require.Equal(testObj, elem)
			counter++
		}
		require.Equal(1, counter)
		require.NoError(*responseErr)
	})
}
