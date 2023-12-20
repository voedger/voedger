/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package pipeline

import (
	"context"
	"time"
)

type WiredOperator[T any] struct {
	name          string
	wctx          IWorkpieceContext
	Stdin         chan T // Stdin is provided by the builder
	Stdout        chan T // Stdout is owned by WiredOperator
	Operator      IOperator
	FlushInterval time.Duration
	ctx           context.Context
	err           IErrorPipeline
	flushCB       OpFuncFlush
}

func WireAsyncOperator(name string, op IAsyncOperator, flushIntvl ...time.Duration) *WiredOperator[IWorkpiece] {
	var flush time.Duration
	if len(flushIntvl) > 0 {
		flush = flushIntvl[0]
	}

	res := &WiredOperator[IWorkpiece]{
		name:          name,
		Stdin:         nil,
		Stdout:        make(chan IWorkpiece, 1),
		Operator:      op,
		FlushInterval: flush,
	}

	flushCB := func(work IWorkpiece) {
		if res.isActive() {
			res.Stdout <- work
		}
	}

	res.flushCB = flushCB
	return res
}

func WireSyncOperator[T any](name string, op ISyncOperator[T]) *WiredOperator[T] {
	return &WiredOperator[T]{
		name:     name,
		Stdin:    nil,
		Stdout:   make(chan T, 1),
		Operator: op,
	}
}

func (wo WiredOperator[T]) isActive() bool {
	return wo.ctx.Err() == nil && wo.err == nil
}

func (wo WiredOperator[T]) forwardIfErrorAsync(work IWorkpiece) bool {
	if work == nil {
		pipelinePanic("nil in puller_async stdin", wo.name, wo.wctx)
	}

	if err, ok := work.(IErrorPipeline); ok {
		wo.Operator.(IAsyncOperator).OnError(wo.ctx, err)
		wo.Stdout <- err
		return true
	}
	return false
}

func (wo WiredOperator[T]) String() string {
	return "operator: " + wo.name
}

func (wo *WiredOperator[T]) NewError(err error, work interface{}, place string) IErrorPipeline {
	ep := errPipeline{
		err:    err,
		work:   work,
		place:  place,
		opName: wo.name,
	}
	wo.err = &ep
	return &ep
}

func (wo *WiredOperator[T]) doAsync(work IWorkpiece) (IWorkpiece, IErrorPipeline) {
	outWork, e := wo.Operator.(IAsyncOperator).DoAsync(wo.ctx, work)
	if e != nil {
		if outWork == nil {
			return nil, wo.NewError(e, work, placeDoAsyncOutWorkIsNil)
		}
		// TODO: p_release(work)?
		return nil, wo.NewError(e, outWork, placeDoAsyncOutWorkNotNil)
	}
	return outWork, nil
}

func (wo *WiredOperator[T]) doSync(_ context.Context, work T) IErrorPipeline {
	e := wo.Operator.(ISyncOperator[T]).DoSync(wo.ctx, work)
	if e != nil {
		return wo.NewError(e, work, placeDoSync)
	}
	return nil
}
