// Copyright (c) 2021-present Voedger Authors.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package pipeline

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var called bool

type syncOpNoBoilerplate struct {
	NOOP
}

type asyncOpNoBoilerplate struct {
	AsyncNOOP
}

type serviceNoBoilerplate struct {
	NOPService
}

func (s *asyncOpNoBoilerplate) Close() {
	called = true
}

func (s *syncOpNoBoilerplate) Close() {
	called = true
}

func (s *serviceNoBoilerplate) Stop() {
	called = true
}

func reset() {
	called = false
}

func TestBasicUsage_NOOP(t *testing.T) {
	require := require.New(t)
	t.Run("sync operator", func(t *testing.T) {
		defer reset()
		ctx := context.Background()
		so := WireSyncOperator("test op", &syncOpNoBoilerplate{})
		p := NewSyncPipeline(ctx, "test", so)
		work := newTestWork()
		require.NoError(p.SendSync(work))
		require.False(called)
		p.Close()
		require.True(called)
	})

	t.Run("async operator", func(t *testing.T) {
		defer reset()
		ctx := context.Background()
		so := WireAsyncOperator("test op", &asyncOpNoBoilerplate{}, time.Second)
		p := NewAsyncPipeline(ctx, "test", so)
		work := newTestWork()
		require.NoError(p.SendAsync(work))
		require.False(called)
		p.Close()
		require.True(called)
	})

	t.Run("IService", func(t *testing.T) {
		defer reset()
		srv := &serviceNoBoilerplate{}
		so := ServiceOperator(srv)
		require.False(called)
		require.NoError(so.DoSync(context.TODO(), nil))
		so.Close()
		require.True(called)
	})
}

func TestBasicUsage_Make(t *testing.T) {
	require := require.New(t)
	t.Run("sync operator", func(t *testing.T) {
		defer reset()
		ctx := context.Background()
		so := WireFunc("test op", func(ctx context.Context, work IWorkpiece) (err error) {
			called = true
			return
		})
		p := NewSyncPipeline(ctx, "test", so)
		work := newTestWork()
		require.False(called)
		require.NoError(p.SendSync(work))
		require.True(called)
		p.Close()
	})

	t.Run("async operator", func(t *testing.T) {
		defer reset()
		ctx := context.Background()
		ao := WireAsyncFunc("test op", func(ctx context.Context, work IWorkpiece) (outWork IWorkpiece, err error) {
			called = true
			return
		})
		p := NewAsyncPipeline(ctx, "test", ao)
		work := newTestWork()
		require.False(called)
		require.NoError(p.SendAsync(work))
		p.Close()
		require.True(called)
	})

	t.Run("IService", func(t *testing.T) {
		defer reset()
		srv := NewService(func(ctx context.Context) {
			called = true
		})
		so := ServiceOperator(srv)
		require.False(called)
		require.False(called)
		require.NoError(so.DoSync(context.TODO(), nil))
		so.Close()
		require.True(called)
	})
}

func TestNOOP_Cover(t *testing.T) {
	require := require.New(t)

	t.Run("NOOPSync", func(t *testing.T) {
		s := &NOOP{}
		require.NoError(s.DoSync(context.TODO(), nil))
		s.Close()
	})

	t.Run("NOOPAsync", func(t *testing.T) {
		as := &AsyncNOOP{}
		as.OnError(context.TODO(), nil)
		w, err := as.DoAsync(context.TODO(), nil)
		require.Nil(w)
		require.NoError(err)
		require.NoError(as.Flush(nil))
		as.Close()
	})

	t.Run("NOService", func(t *testing.T) {
		is := &NOPService{}
		require.NoError(is.Prepare(nil))
		is.Run(context.TODO())
		is.Stop()
	})

	t.Run("make sync operator", func(t *testing.T) {
		s := NewSyncOp[IWorkpiece](nil)
		require.NoError(s.DoSync(context.TODO(), nil))
		s.Close()
	})
	t.Run("make async operator", func(t *testing.T) {
		as := NewAsyncOp[IWorkpiece](nil)
		w, err := as.DoAsync(context.TODO(), nil)
		require.Nil(w)
		require.NoError(err)
		require.NoError(as.Flush(nil))
		as.Close()
	})
	t.Run("make IService", func(t *testing.T) {
		is := NewService(nil)
		require.NoError(is.Prepare(nil))
		is.Run(context.TODO())
		is.Stop()
	})
}
