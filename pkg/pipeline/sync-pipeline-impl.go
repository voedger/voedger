/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package pipeline

import (
	"context"
	"strings"
)

type SyncPipeline struct {
	name string
	wctx IWorkpieceContext
	ctx  context.Context
	// stdin created by pipeline
	stdin chan interface{}
	// stdout points to the Stdout of the last operator
	stdout    chan interface{}
	operators []*WiredOperator
}

func NewSyncPipeline(ctx context.Context, name string, first *WiredOperator, others ...*WiredOperator) ISyncPipeline {
	var pstruct strings.Builder
	pipeline := &SyncPipeline{
		ctx:       ctx,
		name:      name,
		stdin:     make(chan interface{}, 1),
		operators: make([]*WiredOperator, 1),
	}
	checkSyncOperator(first)
	pipeline.operators[0] = first
	first.Stdin = pipeline.stdin
	pipeline.stdout = first.Stdout
	pstruct.WriteString(first.String())
	last := first

	for _, next := range others {
		checkSyncOperator(next)
		next.Stdin = last.Stdout
		pipeline.operators = append(pipeline.operators, next)
		last = next
		pstruct.WriteString(", ")
		pstruct.WriteString(next.String())
	}
	pipeline.stdout = last.Stdout
	pipeline.wctx = NewWorkpieceContext(name, pstruct.String())

	for _, op := range pipeline.operators {
		op.ctx = ctx
		op.wctx = pipeline.wctx
	}
	for _, op := range pipeline.operators {
		go puller_sync(op)
	}
	return pipeline
}

func (p SyncPipeline) DoSync(_ context.Context, work interface{}) (err error) {
	return p.SendSync(work)
}

func (p SyncPipeline) SendSync(work interface{}) (err error) {
	if p.ctx.Err() != nil {
		return p.ctx.Err()
	}
	p.stdin <- work
	outWork := <-p.stdout
	if err, ok := outWork.(error); ok {
		return err
	}
	return nil
}

func (p SyncPipeline) Close() {
	close(p.stdin)
	for range p.stdout {
	}
}

func checkSyncOperator(wo *WiredOperator) {
	if _, ok := wo.Operator.(ISyncOperator); !ok {
		panic("sync pipeline only supports sync operators")
	}
}
