/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package coreutils

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/istructs"
)

func TestRequestSender_BasicUsage(t *testing.T) {
	require := require.New(t)
	requestSender := NewIRequestSender(MockTime, DefaultSendTimeout, func(requestCtx context.Context, request Request, responder IResponder) {
		require.Equal(http.MethodPost, request.Method)
		require.Equal(istructs.WSID(1), request.WSID)
		require.Equal(istructs.PartitionID(2), request.PartitionID)
		require.Equal(map[string][]string{ContentType: {ApplicationJSON}}, request.Header)
		require.Equal(map[string][]string{"param": {"value"}}, request.Query)
		require.Equal("c.sys.CUD", request.Resource)
		require.Equal([]byte("body"), request.Body)
		require.Equal(istructs.AppQName_test1_app1.String(), request.AppQName)
		require.Equal("localhost", request.Host)

		//response must be made in a separate thread
		go func() {
			sender := responder.InitResponse(ResponseMeta{ContentType: ApplicationJSON, StatusCode: http.StatusOK})
			result := map[string]interface{}{
				"fld1": 42,
				"fld2": "str",
			}
			err := sender.Send(result)
			require.NoError(err)
			sender.Close(nil)
		}()
	})

	respCh, respMeta, respErr, err := requestSender.SendRequest(context.Background(), Request{
		Method:      http.MethodPost,
		WSID:        1,
		PartitionID: 2,
		Header: map[string][]string{
			ContentType: {ApplicationJSON},
		},
		Resource: "c.sys.CUD",
		Query: map[string][]string{
			"param": {"value"},
		},
		Body:     []byte("body"),
		AppQName: istructs.AppQName_test1_app1.String(),
		Host:     "localhost",
	})
	require.NoError(err)

	// respCh must be read out
	for elem := range respCh {
		require.Equal(map[string]interface{}{"fld1": 42, "fld2": "str"}, elem)
	}

	// respErr must be checked right after respCh read out
	require.NoError(*respErr)

	require.Equal(ApplicationJSON, respMeta.ContentType)
	require.Equal(http.StatusOK, respMeta.StatusCode)
}

func TestErrorBeforeSend(t *testing.T) {
	
}
