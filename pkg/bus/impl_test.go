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

func TestRequestSender_Streaming_BasicUsage(t *testing.T) {
	require := require.New(t)

	cases := []struct {
		name    string
		handler func(responder IResponder)
	}{
		{
			name: "multi response",
			handler: func(responder IResponder) {
				sender := responder.BeginStreamingResponse(http.StatusOK)
				result := map[string]interface{}{
					"fld1": 42,
					"fld2": "str",
				}
				err := sender.Send(result)
				require.NoError(err)
				sender.Close(nil)
			},
		},
		{
			name: "single response",
			handler: func(responder IResponder) {
				err := responder.Respond(http.StatusOK, map[string]interface{}{
					"fld1": 42,
					"fld2": "str",
				})
				require.NoError(err)
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
			for elem := range respCh {
				require.Equal(map[string]interface{}{"fld1": 42, "fld2": "str"}, elem)
			}

			// respErr must be checked right after respCh read out
			require.NoError(*respErr)

			require.Equal(coreutils.ApplicationJSON, respMeta.ContentType)
			require.Equal(http.StatusOK, respMeta.StatusCode)
		})
	}
}

func TestErrors_Multi(t *testing.T) {
	require := require.New(t)
	t.Run("response timeout", func(t *testing.T) {
		requestSender := NewIRequestSender(coreutils.MockTime, DefaultSendTimeout, func(requestCtx context.Context, request Request, responder IResponder) {
			// wait to start response awaiting in request sender
			time.Sleep(100 * time.Millisecond)
			// force response timeout
			coreutils.MockTime.Add(time.Duration(DefaultSendTimeout))
			sender := responder.BeginStreamingResponse(http.StatusOK)
			sender.Close(nil)
		})

		_, _, _, err := requestSender.SendRequest(context.Background(), Request{})
		require.ErrorIs(err, ErrSendTimeoutExpired)
	})

	t.Run("client disconnect on send response", func(t *testing.T) {
		maySendAfterDisconnect := make(chan interface{})
		sendErrCh := make(chan error)
		clientCtx, disconnectClient := context.WithCancel(context.Background())
		requestSender := NewIRequestSender(coreutils.MockTime, DefaultSendTimeout, func(requestCtx context.Context, request Request, responder IResponder) {
			go func() {
				sender := responder.BeginStreamingResponse(http.StatusOK)
				<-maySendAfterDisconnect
				sendErrCh <- sender.Send("test")
				sender.Close(nil)
			}()
		})

		respCh, _, respErr, err := requestSender.SendRequest(clientCtx, Request{})
		require.NoError(err)
		disconnectClient()
		close(maySendAfterDisconnect)
		err = <-sendErrCh
		require.ErrorIs(err, context.Canceled)
		for range respCh {
		}
		require.NoError(*respErr)
	})

	t.Run("client disconnect on send request", func(t *testing.T) {
		requestHandlerStarted := make(chan interface{})
		sendErrCh := make(chan error)
		clientCtx, disconnectClient := context.WithCancel(context.Background())
		requestSender := NewIRequestSender(coreutils.MockTime, DefaultSendTimeout, func(requestCtx context.Context, request Request, responder IResponder) {
			close(requestHandlerStarted)
			go func() {
				<-clientCtx.Done()
				sender := responder.BeginStreamingResponse(http.StatusOK)
				sendErrCh <- sender.Send("test")
				sender.Close(nil)
			}()
		})

		go func() {
			<-requestHandlerStarted
			disconnectClient()
		}()

		respCh, _, respErr, err := requestSender.SendRequest(clientCtx, Request{})
		require.ErrorIs(err, context.Canceled)
		err = <-sendErrCh
		require.ErrorIs(err, context.Canceled)
		for range respCh {
		}
		require.NoError(*respErr)
	})

	t.Run("no consumer", func(t *testing.T) {
		sendErrCh := make(chan error)
		maySend := make(chan interface{})
		requestSender := NewIRequestSender(coreutils.MockTime, DefaultSendTimeout, func(requestCtx context.Context, request Request, responder IResponder) {
			go func() {
				sender := responder.BeginStreamingResponse(http.StatusOK)
				<-maySend
				sendErrCh <- sender.Send("test")
				sender.Close(nil)
			}()
		})

		respCh, _, respErr, err := requestSender.SendRequest(context.Background(), Request{})
		require.NoError(err)
		close(maySend)

		// sleep to make sure we're in select in Send()
		time.Sleep(100 * time.Millisecond)

		// force send timeout
		coreutils.MockTime.Add(time.Duration(DefaultSendTimeout + SendTimeout(time.Second)))

		err = <-sendErrCh
		require.ErrorIs(err, ErrNoConsumer)
		for range respCh {
		}
		require.NoError(*respErr)
	})
}

func TestErrors_Single(t *testing.T) {
	require := require.New(t)
	var testObject = "test"
	t.Run("response timeout", func(t *testing.T) {
		sendErrCh := make(chan error)
		requestSender := NewIRequestSender(coreutils.MockTime, DefaultSendTimeout, func(requestCtx context.Context, request Request, responder IResponder) {
			// wait to start response awaiting in request sender
			time.Sleep(100 * time.Millisecond)
			// force response timeout
			coreutils.MockTime.Add(time.Duration(DefaultSendTimeout))
			go func() {
				forceTimeoutOnSend()
				sendErrCh <- responder.Respond(http.StatusOK, testObject)
			}()
		})

		_, _, _, err := requestSender.SendRequest(context.Background(), Request{})
		require.ErrorIs(err, ErrSendTimeoutExpired)
		require.ErrorIs(<-sendErrCh, ErrNoConsumer)
	})

	t.Run("client disconnect on send request", func(t *testing.T) {
		requestHandlerStarted := make(chan interface{})
		sendErrCh := make(chan error)
		clientCtx, disconnectClient := context.WithCancel(context.Background())
		requestSender := NewIRequestSender(coreutils.MockTime, DefaultSendTimeout, func(requestCtx context.Context, request Request, responder IResponder) {
			close(requestHandlerStarted)
			go func() {
				<-clientCtx.Done()
				forceTimeoutOnSend()
				sendErrCh <- responder.Respond(http.StatusOK, testObject)
			}()
		})

		go func() {
			<-requestHandlerStarted
			disconnectClient()
		}()

		_, _, respErr, err := requestSender.SendRequest(clientCtx, Request{})
		require.ErrorIs(err, context.Canceled)
		err = <-sendErrCh
		require.ErrorIs(err, context.Canceled)
		require.NoError(*respErr)
	})
}

func TestHandlerPanic(t *testing.T) {
	requestSender := NewIRequestSender(coreutils.MockTime, DefaultSendTimeout, func(requestCtx context.Context, request Request, responder IResponder) {
		panic("test panic")
	})

	_, _, _, err := requestSender.SendRequest(context.Background(), Request{})
	require.ErrorContains(t, err, "test panic")
}

// force trigger timeout case on Send under the hood of responder.Respond
func forceTimeoutOnSend() {
	go func() {
		time.Sleep(100 * time.Millisecond)
		coreutils.MockTime.Add(time.Duration(DefaultSendTimeout))
	}()
}
