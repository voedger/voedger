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

func TestReply(t *testing.T) {
	require := require.New(t)

	type expected struct {
		code  int
		error string
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
					error: `{"sys.Error":{"HTTPStatus":500,"Message":"test error"}}`,
				},
			},
			{
				desc: "ReplyErrf",
				f:    func(responder IResponder) { ReplyErrf(responder, http.StatusAccepted, "test ", "message") },
				expected: expected{
					code:  http.StatusAccepted,
					error: `{"sys.Error":{"HTTPStatus":202,"Message":"test message"}}`,
				},
			},
			{
				desc: "ReplyErrorDef",
				f:    func(responder IResponder) { ReplyErrDef(responder, err, http.StatusAccepted) },
				expected: expected{
					code:  http.StatusAccepted,
					error: `{"sys.Error":{"HTTPStatus":202,"Message":"test error"}}`,
				},
			},
			{
				desc: "SysError",
				f: func(responder IResponder) {
					err := coreutils.SysError{
						HTTPStatus: http.StatusAlreadyReported,
						Message:    "test error",
						Data:       "dddfd",
						QName:      appdef.NewQName("my", "qname"),
					}
					ReplyErrDef(responder, err, http.StatusAccepted)
				},
				expected: expected{
					code:  http.StatusAlreadyReported,
					error: `{"sys.Error":{"HTTPStatus":208,"Message":"test error","QName":"my.qname","Data":"dddfd"}}`,
				},
			},
			{
				desc: "ReplyInternalServerError",
				f: func(responder IResponder) {
					ReplyInternalServerError(responder, "test", err)
				},
				expected: expected{
					code:  http.StatusInternalServerError,
					error: `{"sys.Error":{"HTTPStatus":500,"Message":"test: test error"}}`,
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
				require.Equal(c.expected.error, cmdResp.SysError.ToJSON())
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
					c.f(responder, "test message")
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
}
