/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package pipeline

import (
	"context"
	"fmt"
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
}

func WireAsyncOperator(name string, op IAsyncOperator, flushIntvl ...time.Duration) *WiredOperator {
	var flush time.Duration
	if len(flushIntvl) > 0 {
		flush = flushIntvl[0]
	}
	return &WiredOperator{
		name:          name,
		Stdin:         nil,
		Stdout:        make(chan interface{}, 1),
		Operator:      op,
		FlushInterval: flush,
	}
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

func (wo *WiredOperator) NewError(err error, work interface{}, place string) IErrorPipeline {
	ep := errPipeline{
		err:  fmt.Errorf("[%s/%s] %w", wo.name, place, err),
		work: work,
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

func (wo *WiredOperator) doSync(_ context.Context, work interface{}) IErrorPipeline {
	e := wo.Operator.(ISyncOperator).DoSync(wo.ctx, work)
	if e != nil {
		return wo.NewError(e, work, placeDoSync)
	}
	return nil
}
