/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package pipeline

import (
	"context"
)

type NOOP struct{}

type AsyncNOOP struct{}

type NOPService struct{}

type implISyncOperatorSimple struct {
	NOOP
	doSync func(ctx context.Context, work interface{}) (err error)
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

func (n *NOOP) DoSync(ctx context.Context, work interface{}) (err error) {
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

func (so *implISyncOperatorSimple) DoSync(ctx context.Context, work interface{}) (err error) {
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

// based on ISyncOperator
func WireFunc(name string, doSync func(ctx context.Context, work interface{}) (err error)) *WiredOperator {
	return WireSyncOperator(name, NewSyncOp(doSync))
}

// based on IAsyncOperator
func WireAsyncFunc(name string, doAsync func(ctx context.Context, work IWorkpiece) (outWork IWorkpiece, err error)) *WiredOperator {
	return WireAsyncOperator(name, NewAsyncOp(doAsync))
}

func NewSyncOp(doSync func(ctx context.Context, work interface{}) (err error)) ISyncOperator {
	return &implISyncOperatorSimple{doSync: doSync}
}

func NewAsyncOp(doAsync func(ctx context.Context, work IWorkpiece) (outWork IWorkpiece, err error)) IAsyncOperator {
	return &implAsyncOperatorSimple{doAsync: doAsync}
}

func NewService(run func(ctx context.Context)) IService {
	return &implIServiceSimple{run: run}
}
