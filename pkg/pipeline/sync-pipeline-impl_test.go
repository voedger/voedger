// Copyright (c) 2021-present Voedger Authors.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.
package pipeline

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSyncPipeline_DoSync(t *testing.T) {
	t.Run("Should return unhandled error", func(t *testing.T) {
		pipeline := NewSyncPipeline(context.Background(), "my-pipeline",
			WireFunc("apply-name", opName),
			WireFunc("fail-here", opError),
			WireFunc("passthrough-error", opSex),
		)
		defer pipeline.Close()

		err := pipeline.SendSync(newTestWork())

		require.Error(t, err)
		require.Equal(t, "test failure", err.Error())
		var pErr IErrorPipeline
		require.ErrorAs(t, err, &pErr)
		require.Equal(t, "fail-here", pErr.GetOpName())
		require.Equal(t, "doSync", pErr.GetPlace())
		require.NotNil(t, pErr.GetWork())
	})
	t.Run("Should catch and rethrow error", func(t *testing.T) {
		pipeline := NewSyncPipeline(context.Background(), "my-pipeline",
			WireFunc("apply-name", opName),
			WireFunc("fail-here", opError),
			WireSyncOperator("catch-and-rethrow", mockSyncOp().
				catch(func(err error, work interface{}, context IWorkpieceContext) (newErr error) {
					work.(testwork).slots["error"] = err
					work.(testwork).slots["error-ctx"] = context
					return errors.New("rethrown")
				}).
				create()),
		)
		defer pipeline.Close()

		err := pipeline.SendSync(newTestWork())
		require.Error(t, err)
		var perr IErrorPipeline
		require.ErrorAs(t, err, &perr)
		require.Equal(t, "nested error 'rethrown' while handling 'test failure'", perr.Error())
		require.Equal(t, "catch-and-rethrow", perr.GetOpName())
		require.Equal(t, "catch-onErr", perr.GetPlace())
	})
	t.Run("Should return error on ctx termination", func(t *testing.T) {
		ctx := &testContext{}
		pipeline := NewSyncPipeline(ctx, "my-pipeline",
			WireFunc("apply-name", opName))

		testerr := errors.New("context termination")
		ctx.err = testerr
		err := pipeline.DoSync(ctx, nil)

		require.Equal(t, testerr, err)
	})
	t.Run("Should handle not a workpiece with noop operator", func(t *testing.T) {
		ctx := &testContext{}
		v := &notAWorkpiece{}
		pipeline := NewSyncPipeline(ctx, "my-pipeline",
			WireSyncOperator("noop", &NOOP{}))

		require.NoError(t, pipeline.DoSync(ctx, v))
	})
	t.Run("Should panic on nil work", func(t *testing.T) {
		pipeline := NewSyncPipeline(context.Background(), "my-pipeline",
			WireFunc[IWorkpiece]("panic-onNil", nil))

		require.PanicsWithValue(t, "critical error in operator 'panic-onNil': nil work in processSyncOp. Pipeline 'my-pipeline' [operator: panic-onNil]", func() {
			_ = pipeline.SendSync(nil)
		})
	})
}

func TestSyncPipeline_Close(t *testing.T) {
	pipeline := &SyncPipeline{}

	require.NotPanics(t, func() {
		pipeline.Close()
	})
}

func Test_checkSyncOperator(t *testing.T) {
	t.Run("Should not panic", func(t *testing.T) {
		require.NotPanics(t, func() {
			checkSyncOperator(WireFunc("operator", opAge))
		})
	})

	t.Run("Should panic when operator isn't sync", func(t *testing.T) {
		require.PanicsWithValue(t, "sync pipeline only supports sync operators", func() {
			checkSyncOperator(WireAsyncOperator("operator", NewAsyncOp[IWorkpiece](nil)))
		})
	})
}
