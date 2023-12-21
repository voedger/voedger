/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package pipeline

import (
	"context"
	"strings"
)

type SyncPipeline[T any] struct {
	name string
	wctx IWorkpieceContext
	ctx  context.Context
	// stdin created by pipeline
	stdin chan T
	// stdout points to the Stdout of the last operator
	stdout    chan T
	operators []*WiredOperator[T]
}

func NewSyncPipeline[T any](ctx context.Context, name string, first *WiredOperator[T], others ...*WiredOperator[T]) ISyncPipeline[T] {
	var pstruct strings.Builder
	pipeline := &SyncPipeline[T]{
		ctx:       ctx,
		name:      name,
		stdin:     make(chan T, 1),
		operators: make([]*WiredOperator[T], 1),
	}
	checkSyncOperator[T](first)
	pipeline.operators[0] = first
	first.Stdin = pipeline.stdin
	pipeline.stdout = first.Stdout
	pstruct.WriteString(first.String())
	last := first

	for _, next := range others {
		checkSyncOperator[T](next)
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

// FIXME pointer?
func (p SyncPipeline[T]) DoSync(_ context.Context, work T) (err error) {
	return p.SendSync(work)
}

func (p SyncPipeline[T]) SendSync(work T) (err error) {
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

func (p SyncPipeline[T]) Close() {
	close(p.stdin)
	for range p.stdout {
	}
}

func checkSyncOperator[T any](wo *WiredOperator[T]) {
	if _, ok := wo.Operator.(ISyncOperator[T]); !ok {
		panic("sync pipeline only supports sync operators")
	}
}
