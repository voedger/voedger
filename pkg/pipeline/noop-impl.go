// Copyright (c) 2021-present Voedger Authors.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.
package pipeline

import (
	"context"
)

type NOOP struct{}

type AsyncNOOP struct{}

type NOPService struct{}

type implISyncOperatorSimple struct {
	NOOP
	doSync func(ctx context.Context, work IWorkpiece) (err error)
}

type implAsyncOperatorSimple struct {
	AsyncNOOP
	doAsync func(ctx context.Context, work IWorkpiece) (outWork IWorkpiece, err error)
}

type implIServiceSimple struct {
	NOPService
	run func(ctx context.Context)
}

func (n *NOOP) Close() {}

func (n *NOOP) DoSync(_ context.Context, _ IWorkpiece) (err error) {
	return
}

func (n *AsyncNOOP) Close() {}

func (n *AsyncNOOP) DoAsync(ctx context.Context, work IWorkpiece) (outWork IWorkpiece, err error) {
	return
}

func (n *AsyncNOOP) Flush(callback OpFuncFlush) (err error) {
	return
}

func (n *AsyncNOOP) OnError(ctx context.Context, err error) {}

func (n *NOPService) Prepare(work interface{}) (err error) {
	return
}

func (n *NOPService) Run(ctx context.Context) {}

func (n *NOPService) Stop() {}

func (so *implISyncOperatorSimple) DoSync(ctx context.Context, work IWorkpiece) (err error) {
	if so.doSync != nil {
		return so.doSync(ctx, work)
	}
	return
}

func (ao *implAsyncOperatorSimple) DoAsync(ctx context.Context, work IWorkpiece) (outWork IWorkpiece, err error) {
	if ao.doAsync != nil {
		return ao.doAsync(ctx, work)
	}
	return
}

func (s *implIServiceSimple) Run(ctx context.Context) {
	if s.run != nil {
		s.run(ctx)
	}
}

// WireFunc creates a WiredOperator with a sync operator.
// T can be any type that implements IWorkpiece, allowing typed workpiece access.
func WireFunc[T IWorkpiece](name string, doSync func(ctx context.Context, work T) (err error)) *WiredOperator {
	return WireSyncOperator(name, NewSyncOp(doSync))
}

// WireAsyncFunc creates a WiredOperator with an async operator.
// T can be any type that implements IWorkpiece, allowing typed workpiece access.
func WireAsyncFunc[T IWorkpiece](name string, doAsync func(ctx context.Context, work T) (outWork IWorkpiece, err error)) *WiredOperator {
	return WireAsyncOperator(name, NewAsyncOp(doAsync))
}

// NewSyncOp creates a sync operator from a function.
// T can be any type that implements IWorkpiece, allowing typed workpiece access.
func NewSyncOp[T IWorkpiece](doSync func(ctx context.Context, work T) (err error)) ISyncOperator {
	if doSync == nil {
		return &implISyncOperatorSimple{}
	}
	return &implISyncOperatorSimple{doSync: func(ctx context.Context, work IWorkpiece) error {
		var typedWork T
		if work != nil {
			typedWork = work.(T)
		}
		return doSync(ctx, typedWork)
	}}
}

// NewAsyncOp creates an async operator from a function.
// T can be any type that implements IWorkpiece, allowing typed workpiece access.
func NewAsyncOp[T IWorkpiece](doAsync func(ctx context.Context, work T) (outWork IWorkpiece, err error)) IAsyncOperator {
	if doAsync == nil {
		return &implAsyncOperatorSimple{}
	}
	return &implAsyncOperatorSimple{doAsync: func(ctx context.Context, work IWorkpiece) (IWorkpiece, error) {
		var typedWork T
		if work != nil {
			typedWork = work.(T)
		}
		return doAsync(ctx, typedWork)
	}}
}

func NewService(run func(ctx context.Context)) IService {
	return &implIServiceSimple{run: run}
}
