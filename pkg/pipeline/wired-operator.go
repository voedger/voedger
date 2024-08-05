// Copyright (c) 2021-present Voedger Authors.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package pipeline

import (
	"context"
	"time"
)

type WiredOperator struct {
	name          string
	wctx          IWorkpieceContext
	Stdin         chan interface{} // Stdin is provided by the builder
	Stdout        chan interface{} // Stdout is owned by WiredOperator
	Operator      IOperator
	FlushInterval time.Duration
	ctx           context.Context
	err           IErrorPipeline
	flushCB       OpFuncFlush
}

func WireAsyncOperator(name string, op IAsyncOperator, flushIntvl ...time.Duration) *WiredOperator {
	var flush time.Duration
	if len(flushIntvl) > 0 {
		flush = flushIntvl[0]
	}

	res := &WiredOperator{
		name:          name,
		Stdin:         nil,
		Stdout:        make(chan interface{}, 1),
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

func WireSyncOperator(name string, op ISyncOperator) *WiredOperator {
	return &WiredOperator{
		name:     name,
		Stdin:    nil,
		Stdout:   make(chan interface{}, 1),
		Operator: op,
	}
}

func (wo WiredOperator) isActive() bool {
	return wo.ctx.Err() == nil && wo.err == nil
}

func (wo WiredOperator) forwardIfErrorAsync(work IWorkpiece) bool {
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

func (wo WiredOperator) String() string {
	return "operator: " + wo.name
}

func (wo *WiredOperator) NewError(err error, work IWorkpiece, place string) IErrorPipeline {
	ep := errPipeline{
		err:    err,
		work:   work,
		place:  place,
		opName: wo.name,
	}
	wo.err = &ep
	return &ep
}

func (wo *WiredOperator) doAsync(work IWorkpiece) (IWorkpiece, IErrorPipeline) {
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

func (wo *WiredOperator) doSync(_ context.Context, work IWorkpiece) IErrorPipeline {
	e := wo.Operator.(ISyncOperator).DoSync(wo.ctx, work)
	if e != nil {
		return wo.NewError(e, work, placeDoSync)
	}
	return nil
}
