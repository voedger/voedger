/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package queryprocessor

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/coreutils"
)

func TestSendToBusOperator_DoAsync(t *testing.T) {
	require := require.New(t)
	errCh := make(chan error, 1)
	requestSender := bus.NewIRequestSender(coreutils.MockTime, bus.GetTestSendTimeout(), func(requestCtx context.Context, request bus.Request, responder bus.IResponder) {
		go func() {
			operator := SendToBusOperator{
				responder: responder,
				metrics:   &testMetrics{},
				errCh:     errCh,
			}
			work := rowsWorkpiece{outputRow: &testOutputRow{values: []interface{}{"hello world"}}}

			outWork, err := operator.DoAsync(context.Background(), work)
			_, _ = operator.DoAsync(context.Background(), work)

			require.Equal(work, outWork)
			require.NoError(err)

			operator.sender.(bus.IResponseSenderCloseable).Close(nil)
		}()
	})

	respCh, respMeta, respErr, err := requestSender.SendRequest(context.Background(), bus.Request{})
	require.NoError(err)
	require.Equal(coreutils.ApplicationJSON, respMeta.ContentType)
	require.Equal(http.StatusOK, respMeta.StatusCode)
	result := []string{}
	for elem := range respCh {
		result = append(result, elem.([]interface{})[0].(string))
	}
	require.NoError(*respErr)
	require.EqualValues([]string{"hello world", "hello world"}, result)

}

func TestSendToBusOperator_OnError(t *testing.T) {
	require := require.New(t)
	errCh := make(chan error, 1)
	testError := errors.New("test error")
	operator := SendToBusOperator{
		metrics: &testMetrics{},
		errCh:   errCh,
	}

	operator.OnError(context.Background(), testError)

	select {
	case err := <-errCh:
		require.ErrorIs(testError, err)
	default:
		t.Fail()
	}
}
