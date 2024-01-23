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

func TestFilterOperator_DoAsync(t *testing.T) {
	emptyWorkpiece := func() pipeline.IWorkpiece {
		return rowsWorkpiece{
			outputRow: &outputRow{
				keyToIdx: map[string]int{rootDocument: 0},
				values:   []interface{}{[]IOutputRow{&outputRow{}}},
			},
		}
	}
	t.Run("Should filter and release workpiece", func(t *testing.T) {
		require := require.New(t)
		release := false
		workpiece := testWorkpiece{
			outputRow: &outputRow{
				keyToIdx: map[string]int{rootDocument: 0},
				values:   []interface{}{[]IOutputRow{&outputRow{}}},
			},
			release: func() { release = true },
		}
		operator := FilterOperator{
			filters: []IFilter{testFilter{match: false}},
			metrics: &testMetrics{},
		}

		work, err := operator.DoAsync(context.Background(), workpiece)
		require.NoError(err)

		require.Nil(work)
		require.True(release)
	})
	t.Run("Should not filter", func(t *testing.T) {
		operator := FilterOperator{
			filters: []IFilter{testFilter{match: true}},
			metrics: &testMetrics{},
		}

		work, err := operator.DoAsync(context.Background(), emptyWorkpiece())
		require.NoError(t, err)

		require.NotNil(t, work)
	})
	t.Run("Should return error when ctx was cancelled", func(t *testing.T) {
		require := require.New(t)
		operator := FilterOperator{
			filters: []IFilter{nil},
			metrics: &testMetrics{},
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		work, err := operator.DoAsync(ctx, emptyWorkpiece())

		require.Nil(work)
		require.Equal("context canceled", err.Error())
	})
	t.Run("Should return error from filter", func(t *testing.T) {
		require := require.New(t)
		operator := FilterOperator{
			filters: []IFilter{testFilter{match: false, err: ErrWrongType}},
			metrics: &testMetrics{},
		}

		work, err := operator.DoAsync(context.Background(), emptyWorkpiece())

		require.ErrorIs(err, ErrWrongType)
		require.Nil(work)
	})
}
