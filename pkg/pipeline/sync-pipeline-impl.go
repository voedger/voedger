/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

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

func (p SyncPipeline) DoSync(_ context.Context, work interface{}) (err error) {
	return p.SendSync(work)
}

func (p *SyncPipeline) SendSync(work interface{}) (err error) {
	if p.ctx.Err() != nil {
		return p.ctx.Err()
	}
	//p.stdin <- work
	for _, op := range p.operators {
		work = processOp(op, work)
	}
	if err, ok := work.(error); ok {
		return err
	}
	return nil
}

func processOp(wo *WiredOperator, work interface{}) interface{} {
	//work := <-wo.Stdin

	if work == nil {
		pipelinePanic("nil in puller_sync stdin", wo.name, wo.wctx)
	}
	if err, ok := work.(IErrorPipeline); ok {
		if catch, ok := wo.Operator.(ICatch); ok {
			if newerr := catch.OnErr(err, err.GetWork(), wo.wctx); newerr != nil {
				//wo.Stdout <- wo.NewError(fmt.Errorf("nested error '%w' while handling '%w'", newerr, err), err.GetWork(), placeCatchOnErr)
				return wo.NewError(fmt.Errorf("nested error '%w' while handling '%w'", newerr, err), err.GetWork(), placeCatchOnErr)
			}
		} else {
			//wo.Stdout <- err
			return err
		}
		work = err.GetWork() // restore from error
	}

	err := wo.doSync(wo.ctx, work)

	if err != nil {
		//wo.Stdout <- err
		return err
	} else {
		//wo.Stdout <- work
		return work
	}
}

func (p SyncPipeline) Close() {
	//close(p.stdin)
	for _, operator := range p.operators {
		operator.Operator.Close()
	}
}

func checkSyncOperator(wo *WiredOperator) {
	if _, ok := wo.Operator.(ISyncOperator); !ok {
		panic("sync pipeline only supports sync operators")
	}
}
