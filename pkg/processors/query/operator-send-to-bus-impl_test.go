/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package queryprocessor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSendToBusOperator_DoAsync(t *testing.T) {
	require := require.New(t)
	counter := 0
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
	}
	work := rowsWorkpiece{outputRow: &testOutputRow{values: []interface{}{"hello world"}}}

	outWork, err := operator.DoAsync(context.Background(), work)
	_, _ = operator.DoAsync(context.Background(), work)

	require.Equal(work, outWork)
	require.NoError(err)
	require.Equal(1, counter)
}
