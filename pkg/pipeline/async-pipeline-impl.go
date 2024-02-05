// Copyright (c) 2021-present Voedger Authors.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package pipeline

import (
	"context"
	"strings"
)

type AsyncPipeline struct {
	name string
	wctx IWorkpieceContext
	ctx  context.Context
	// stdin created by pipeline
	stdin chan interface{}
	// stdout points to the Stdout of the last operator
	stdout    chan interface{}
	operators []*WiredOperator
}

func NewAsyncPipeline(ctx context.Context, name string, first *WiredOperator, others ...*WiredOperator) IAsyncPipeline {
	var pstruct strings.Builder
	pipeline := &AsyncPipeline{
		ctx:       ctx,
		name:      name,
		stdin:     make(chan interface{}, 1),
		operators: make([]*WiredOperator, 1),
	}
	pipeline.operators[0] = first
	first.Stdin = pipeline.stdin
	pipeline.stdout = first.Stdout
	pstruct.WriteString(first.String())
	last := first

	others = append(others, releaser())
	for _, next := range others {
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
		if _, ok := op.Operator.(IAsyncOperator); ok {
			go puller_async(op)
		} else {
			panic("WiredOperator<ISyncOperator> not allowed in async pipeline")
		}
	}
	return pipeline
}

func (p AsyncPipeline) SendAsync(work IWorkpiece) (err error) {
	if p.ctx.Err() != nil {
		return p.ctx.Err()
	}
	select {
	case p.stdin <- work:
		return nil
	case item := <-p.stdout:
		return item.(error) // only error is possible after releaser
	}
}

func (p AsyncPipeline) Close() {
	close(p.stdin)
	for range p.stdout {
	}
}

func releaser() *WiredOperator {
	return WireAsyncOperator(
		"releaser",
		NewAsyncOp(func(ctx context.Context, work IWorkpiece) (outWork IWorkpiece, err error) {
			work.Release()
			return nil, nil
		}))
}
