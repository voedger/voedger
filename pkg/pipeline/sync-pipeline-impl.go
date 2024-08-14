// Copyright (c) 2021-present Voedger Authors.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package pipeline

import (
	"context"
	"fmt"
	"strings"
)

type SyncPipeline struct {
	name      string
	wctx      IWorkpieceContext
	ctx       context.Context
	operators []*WiredOperator
}

func NewSyncPipeline(ctx context.Context, name string, first *WiredOperator, others ...*WiredOperator) ISyncPipeline {
	var pstruct strings.Builder
	pipeline := &SyncPipeline{
		ctx:       ctx,
		name:      name,
		operators: make([]*WiredOperator, 1),
	}
	checkSyncOperator(first)
	pipeline.operators[0] = first
	pstruct.WriteString(first.String())

	for _, next := range others {
		checkSyncOperator(next)
		pipeline.operators = append(pipeline.operators, next)
		pstruct.WriteString(", ")
		pstruct.WriteString(next.String())
	}
	pipeline.wctx = NewWorkpieceContext(name, pstruct.String())

	for _, op := range pipeline.operators {
		op.ctx = ctx
		op.wctx = pipeline.wctx
	}
	return pipeline
}

func (p *SyncPipeline) DoSync(_ context.Context, work IWorkpiece) (err error) {
	return p.SendSync(work)
}

func (p *SyncPipeline) SendSync(work IWorkpiece) (err error) {
	if p.ctx.Err() != nil {
		return p.ctx.Err()
	}
	for _, op := range p.operators {
		work = processSyncOp(op, work)
	}
	if err, ok := work.(error); ok {
		return err
	}
	return nil
}

func processSyncOp(wo *WiredOperator, work IWorkpiece) IWorkpiece {
	if work == nil {
		pipelinePanic("nil work in processSyncOp", wo.name, wo.wctx)
	}
	if err, ok := work.(IErrorPipeline); ok {
		if catch, ok := wo.Operator.(ICatch); ok {
			if newerr := catch.OnErr(err, err.GetWork(), wo.wctx); newerr != nil {
				return wo.NewError(fmt.Errorf("nested error '%w' while handling '%w'", newerr, err), err.GetWork(), placeCatchOnErr)
			}
		} else {
			return err
		}
		work = err.GetWork() // restore from error
	}

	err := wo.doSync(wo.ctx, work)

	if err != nil {
		return err
	}
	return work
}

func (p *SyncPipeline) Close() {
	for _, operator := range p.operators {
		operator.Operator.Close()
	}
}

func checkSyncOperator(wo *WiredOperator) {
	if _, ok := wo.Operator.(ISyncOperator); !ok {
		panic("sync pipeline only supports sync operators")
	}
}
