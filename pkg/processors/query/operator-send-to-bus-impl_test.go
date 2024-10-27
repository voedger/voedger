/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package queryprocessor

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSendToBusOperator_DoAsync(t *testing.T) {
	require := require.New(t)
	counter := 0
	errCh := make(chan error, 1)
	operator := SendToBusOperator{
		rs: testResultSenderClosable{
			sendElement: func(name string, element interface{}) (err error) {
				require.Equal("hello world", element.([]interface{})[0])
				return nil
			},
			startArraySection: func(sectionType string, path []string) {
				counter++
			},
		},
		metrics: &testMetrics{},
		errCh:   errCh,
	}
	work := rowsWorkpiece{outputRow: &testOutputRow{values: []interface{}{"hello world"}}}

	outWork, err := operator.DoAsync(context.Background(), work)
	_, _ = operator.DoAsync(context.Background(), work)

	require.Equal(work, outWork)
	require.NoError(err)
	require.Equal(1, counter)
}

func TestSendToBusOperator_OnError(t *testing.T) {
	require := require.New(t)
	errCh := make(chan error, 1)
	testError := errors.New("test error")
	operator := SendToBusOperator{
		rs: testResultSenderClosable{
			sendElement: func(name string, element interface{}) (err error) {
				return testError
			},
			startArraySection: func(sectionType string, path []string) {},
		},
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
