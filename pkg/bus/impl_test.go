/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package bus

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/goutils/httpu"
	"github.com/voedger/voedger/pkg/goutils/testingu"
	"github.com/voedger/voedger/pkg/istructs"
)

func TestRequestSender_ApiArray_BasicUsage(t *testing.T) {
	require := require.New(t)

	cases := []struct {
		name    string
		handler func(responder IResponder)
	}{
		{
			name: "api array response",
			handler: func(responder IResponder) {
				respWriter := responder.StreamJSON(http.StatusOK)
				result := map[string]interface{}{
					"fld1": 42,
					"fld2": "str",
				}
				err := respWriter.Write(result)
				require.NoError(err)

				result = map[string]interface{}{
					"fld1": 43,
					"fld2": "str1",
				}
				err = respWriter.Write(result)
				require.NoError(err)
				respWriter.Close(nil)
			},
		},
		{
			name: "custom response",
			handler: func(responder IResponder) {
				respWriter := responder.StreamJSON(http.StatusOK)
				result := map[string]interface{}{
					"fld1": 42,
					"fld2": "str",
				}
				err := respWriter.Write(result)
				require.NoError(err)

				result = map[string]interface{}{
					"fld1": 43,
					"fld2": "str1",
				}
				err = respWriter.Write(result)
				require.NoError(err)
				respWriter.Close(nil)
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			requestSender := NewIRequestSender(testingu.MockTime, func(requestCtx context.Context, request Request, responder IResponder) {
				require.Equal(http.MethodPost, request.Method)
				require.Equal(istructs.WSID(1), request.WSID)
				require.Equal(map[string]string{httpu.ContentType: httpu.ContentType_ApplicationJSON}, request.Header)
				require.Equal(map[string]string{"param": "value"}, request.Query)
				require.Equal("c.sys.CUD", request.Resource)
				require.Equal([]byte("body"), request.Body)
				require.Equal(istructs.AppQName_test1_app1, request.AppQName)
				require.Equal("localhost", request.Host)

				//response must be made in a separate thread
				go c.handler(responder)
			})

			respCh, respMeta, respErr, err := requestSender.SendRequest(context.Background(), Request{
				Method: http.MethodPost,
				WSID:   1,
				Header: map[string]string{
					httpu.ContentType: httpu.ContentType_ApplicationJSON,
				},
				Resource: "c.sys.CUD",
				Query: map[string]string{
					"param": "value",
				},
				Body:     []byte("body"),
				AppQName: istructs.AppQName_test1_app1,
				Host:     "localhost",
			})
			require.NoError(err)

			// respCh must be read out
			counter := 0
			for elem := range respCh {
				switch counter {
				case 0:
					require.Equal(map[string]interface{}{"fld1": 42, "fld2": "str"}, elem)
				case 1:
					require.Equal(map[string]interface{}{"fld1": 43, "fld2": "str1"}, elem)
				default:
					t.Fail()
				}
				counter++
			}

			// respErr must be checked right after respCh read out
			require.NoError(*respErr)

			require.Equal(httpu.ContentType_ApplicationJSON, respMeta.ContentType)
			require.Equal(http.StatusOK, respMeta.StatusCode)
		})
	}
}

func TestErrors(t *testing.T) {
	require := require.New(t)

	t.Run("client disconnect on send response", func(t *testing.T) {
		maySendAfterDisconnect := make(chan interface{})
		writeErrCh := make(chan error)
		clientCtx, disconnectClient := context.WithCancel(context.Background())
		requestSender := NewIRequestSender(testingu.MockTime, func(requestCtx context.Context, request Request, responder IResponder) {
			go func() {
				respWriter := responder.StreamJSON(http.StatusOK)
				<-maySendAfterDisconnect
				writeErrCh <- respWriter.Write("test")
				respWriter.Close(nil)
			}()
		})

		respCh, _, respErr, err := requestSender.SendRequest(clientCtx, Request{})
		require.NoError(err)
		disconnectClient()
		close(maySendAfterDisconnect)
		err = <-writeErrCh
		require.ErrorIs(err, context.Canceled)
		for range respCh {
		}
		require.NoError(*respErr)
	})

	t.Run("client disconnect on send request", func(t *testing.T) {
		requestHandlerStarted := make(chan interface{})
		writeErrCh := make(chan error)
		clientCtx, disconnectClient := context.WithCancel(context.Background())
		requestSender := NewIRequestSender(testingu.MockTime, func(requestCtx context.Context, request Request, responder IResponder) {
			close(requestHandlerStarted)
			go func() {
				<-clientCtx.Done()
				respWriter := responder.StreamJSON(http.StatusOK)
				writeErrCh <- respWriter.Write("test")
				respWriter.Close(nil)
			}()
		})

		go func() {
			<-requestHandlerStarted
			disconnectClient()
		}()

		respCh, _, respErr, err := requestSender.SendRequest(clientCtx, Request{})
		require.ErrorIs(err, context.Canceled)
		err = <-writeErrCh
		require.ErrorIs(err, context.Canceled)
		for range respCh {
		}
		require.NoError(*respErr)
	})

	t.Run("send response timeout", func(t *testing.T) {
		writeErrCh := make(chan error)
		firstWriteDone := make(chan interface{})
		mayWrite2nd := make(chan interface{})
		requestSender := NewIRequestSender(testingu.MockTime, func(requestCtx context.Context, request Request, responder IResponder) {
			go func() {
				respWriter := responder.StreamJSON(http.StatusOK)
				_ = respWriter.Write("test") // first succeed because chan buf is 1
				close(firstWriteDone)
				<-mayWrite2nd
				writeErrCh <- respWriter.Write("test")
				respWriter.Close(nil)
			}()
		})

		respCh, _, respErr, err := requestSender.SendRequest(context.Background(), Request{})
		require.NoError(err)
		<-firstWriteDone

		testingu.MockTime.FireNextTimerImmediately()

		close(mayWrite2nd)

		err = <-writeErrCh
		require.ErrorIs(err, ErrSendResponseTimeout)
		for range respCh {
		}
		require.NoError(*respErr)
	})
}

func TestPanicOnBeginResponseAgain(t *testing.T) {
	require := require.New(t)
	t.Run("api array response", func(t *testing.T) {
		requestSender := NewIRequestSender(testingu.MockTime, func(requestCtx context.Context, request Request, responder IResponder) {
			respWriter := responder.StreamJSON(http.StatusOK)
			require.Panics(func() {
				responder.StreamJSON(http.StatusOK)
			})
			require.Panics(func() { responder.Respond(ResponseMeta{}, nil) })
			respWriter.Close(nil)
		})

		_, _, _, err := requestSender.SendRequest(context.Background(), Request{})
		require.NoError(err)
	})

	t.Run("respond", func(t *testing.T) {
		requestSender := NewIRequestSender(testingu.MockTime, func(requestCtx context.Context, request Request, responder IResponder) {
			err := responder.Respond(ResponseMeta{ContentType: httpu.ContentType_ApplicationJSON, StatusCode: http.StatusOK}, nil)
			require.NoError(err)
			require.Panics(func() {
				responder.StreamJSON(http.StatusOK)
			})
			require.Panics(func() { responder.Respond(ResponseMeta{}, nil) })
		})

		_, _, _, err := requestSender.SendRequest(context.Background(), Request{})
		require.NoError(err)
	})
}

func TestHandlerPanic(t *testing.T) {
	requestSender := NewIRequestSender(testingu.MockTime, func(requestCtx context.Context, request Request, responder IResponder) {
		panic("test panic")
	})

	_, _, _, err := requestSender.SendRequest(context.Background(), Request{})
	require.ErrorContains(t, err, "test panic")
}
