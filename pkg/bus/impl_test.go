/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package bus

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/coreutils"
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
				respWriter := responder.InitResponse(http.StatusOK)
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
				respWriter := responder.InitResponse(http.StatusOK)
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
			requestSender := NewIRequestSender(coreutils.MockTime, DefaultSendTimeout, func(requestCtx context.Context, request Request, responder IResponder) {
				require.Equal(http.MethodPost, request.Method)
				require.Equal(istructs.WSID(1), request.WSID)
				require.Equal(map[string]string{coreutils.ContentType: coreutils.ApplicationJSON}, request.Header)
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
					coreutils.ContentType: coreutils.ApplicationJSON,
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
				if counter == 0 {
					require.Equal(map[string]interface{}{"fld1": 42, "fld2": "str"}, elem)
				} else if counter == 1 {
					require.Equal(map[string]interface{}{"fld1": 43, "fld2": "str1"}, elem)
				} else {
					t.Fail()
				}
				counter++
			}

			// respErr must be checked right after respCh read out
			require.NoError(*respErr)

			require.Equal(coreutils.ApplicationJSON, respMeta.ContentType)
			require.Equal(http.StatusOK, respMeta.StatusCode)
		})
	}
}

func TestErrors(t *testing.T) {
	require := require.New(t)
	t.Run("response timeout ", func(t *testing.T) {
		requestSender := NewIRequestSender(coreutils.MockTime, DefaultSendTimeout, func(requestCtx context.Context, request Request, responder IResponder) {
			// wait to start response awaiting in request sender
			time.Sleep(100 * time.Millisecond)
			// force response timeout
			coreutils.MockTime.Add(time.Duration(DefaultSendTimeout))
			respWriter := responder.InitResponse(http.StatusOK)
			respWriter.Close(nil)
		})

		_, _, _, err := requestSender.SendRequest(context.Background(), Request{})
		require.ErrorIs(err, ErrSendTimeoutExpired)
	})

	t.Run("client disconnect on send response", func(t *testing.T) {
		maySendAfterDisconnect := make(chan interface{})
		writeErrCh := make(chan error)
		clientCtx, disconnectClient := context.WithCancel(context.Background())
		requestSender := NewIRequestSender(coreutils.MockTime, DefaultSendTimeout, func(requestCtx context.Context, request Request, responder IResponder) {
			go func() {
				respWriter := responder.InitResponse(http.StatusOK)
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
		requestSender := NewIRequestSender(coreutils.MockTime, DefaultSendTimeout, func(requestCtx context.Context, request Request, responder IResponder) {
			close(requestHandlerStarted)
			go func() {
				<-clientCtx.Done()
				respWriter := responder.InitResponse(http.StatusOK)
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

	t.Run("no consumer", func(t *testing.T) {
		writeErrCh := make(chan error)
		maySend := make(chan interface{})
		requestSender := NewIRequestSender(coreutils.MockTime, DefaultSendTimeout, func(requestCtx context.Context, request Request, responder IResponder) {
			go func() {
				respWriter := responder.InitResponse(http.StatusOK)
				<-maySend
				_ = respWriter.Write("test") // first succeed because chan buf is 1
				writeErrCh <- respWriter.Write("test")
				respWriter.Close(nil)
			}()
		})

		respCh, _, respErr, err := requestSender.SendRequest(context.Background(), Request{})
		require.NoError(err)
		close(maySend)

		// sleep to make sure we're in select in Send()
		time.Sleep(100 * time.Millisecond)

		// force send timeout
		coreutils.MockTime.Add(time.Duration(DefaultSendTimeout + SendTimeout(time.Second)))

		err = <-writeErrCh
		require.ErrorIs(err, ErrNoConsumer)
		for range respCh {
		}
		require.NoError(*respErr)
	})
}

func TestPanicOnBeginResponseAgain(t *testing.T) {
	require := require.New(t)
	t.Run("api array response", func(t *testing.T) {
		requestSender := NewIRequestSender(coreutils.MockTime, DefaultSendTimeout, func(requestCtx context.Context, request Request, responder IResponder) {
			respWriter := responder.InitResponse(http.StatusOK)
			require.Panics(func() {
				responder.InitResponse(http.StatusOK)
			})
			require.Panics(func() { responder.Respond(ResponseMeta{}, nil) })
			respWriter.Close(nil)
		})

		_, _, _, err := requestSender.SendRequest(context.Background(), Request{})
		require.NoError(err)
	})

	t.Run("respond", func(t *testing.T) {
		requestSender := NewIRequestSender(coreutils.MockTime, DefaultSendTimeout, func(requestCtx context.Context, request Request, responder IResponder) {
			responder.Respond(ResponseMeta{ContentType: coreutils.ApplicationJSON, StatusCode: http.StatusOK}, nil)
			require.Panics(func() {
				responder.InitResponse(http.StatusOK)
			})
			require.Panics(func() { responder.Respond(ResponseMeta{}, nil) })
		})

		_, _, _, err := requestSender.SendRequest(context.Background(), Request{})
		require.NoError(err)
	})
}

func TestHandlerPanic(t *testing.T) {
	requestSender := NewIRequestSender(coreutils.MockTime, DefaultSendTimeout, func(requestCtx context.Context, request Request, responder IResponder) {
		panic("test panic")
	})

	_, _, _, err := requestSender.SendRequest(context.Background(), Request{})
	require.ErrorContains(t, err, "test panic")
}
