/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package queryprocessor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/pipeline"
)

func TestCounterOperator_DoAsync(t *testing.T) {
	tests := []struct {
		startFrom      int64
		count          int64
		name           string
		first          bool
		second         bool
		third          bool
		fourth         bool
		fifth          bool
		sixth          bool
		releaseCounter int64
	}{
		{
			name:           "Should return all workpieces",
			startFrom:      0,
			count:          0,
			first:          true,
			second:         true,
			third:          true,
			fourth:         true,
			fifth:          true,
			sixth:          true,
			releaseCounter: 0,
		},
		{
			name:           "Should skip 2 workpieces then return 2 workpieces",
			startFrom:      2,
			count:          2,
			first:          false,
			second:         false,
			third:          true,
			fourth:         true,
			fifth:          false,
			sixth:          false,
			releaseCounter: 4,
		},
		{
			name:           "Should skip 3 workpieces then return remaining workpieces",
			startFrom:      3,
			count:          0,
			first:          false,
			second:         false,
			third:          false,
			fourth:         true,
			fifth:          true,
			sixth:          true,
			releaseCounter: 3,
		},
	}
	outWorkIsPresent := func(outWork pipeline.IWorkpiece, _ error) bool {
		return outWork != nil
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			releaseCounter := int64(0)
			work := testWorkpiece{release: func() {
				releaseCounter++
			}}
			operator := newCounterOperator(test.startFrom, test.count, &testMetrics{})

			require.Equal(t, test.first, outWorkIsPresent(operator.DoAsync(context.Background(), work)))
			require.Equal(t, test.second, outWorkIsPresent(operator.DoAsync(context.Background(), work)))
			require.Equal(t, test.third, outWorkIsPresent(operator.DoAsync(context.Background(), work)))
			require.Equal(t, test.fourth, outWorkIsPresent(operator.DoAsync(context.Background(), work)))
			require.Equal(t, test.fifth, outWorkIsPresent(operator.DoAsync(context.Background(), work)))
			require.Equal(t, test.sixth, outWorkIsPresent(operator.DoAsync(context.Background(), work)))
			require.Equal(t, test.releaseCounter, releaseCounter)
		})
	}
}
