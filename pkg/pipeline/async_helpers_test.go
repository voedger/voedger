// Copyright (c) 2021-present Voedger Authors.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.
// @author Maxim Geraskin

package pipeline

import "context"

type userEntry struct {
	name string
	role string
}

func (w userEntry) Release() {}

type funcFlush func(callback OpFuncFlush) (err error)
type funcDoAsync func(ctx context.Context, work IWorkpiece) (outWork IWorkpiece, err error)

type mockAsyncOperatorBuilder struct {
	fDoAsync funcDoAsync
	fFlush   funcFlush
	fClose   funcClose
	fOnError funcOnError
}

func (b *mockAsyncOperatorBuilder) doAsync(f funcDoAsync) *mockAsyncOperatorBuilder {
	b.fDoAsync = f
	return b
}

func (b *mockAsyncOperatorBuilder) flush(f funcFlush) *mockAsyncOperatorBuilder {
	b.fFlush = f
	return b
}

func (b *mockAsyncOperatorBuilder) onError(f funcOnError) *mockAsyncOperatorBuilder {
	b.fOnError = f
	return b
}

func (b *mockAsyncOperatorBuilder) create() IAsyncOperator {
	return &mockedAsyncOperator{
		fDoAsync: b.fDoAsync,
		fFlush:   b.fFlush,
		fClose:   b.fClose,
		fOnError: b.fOnError,
	}
}

func mockAsyncOp() *mockAsyncOperatorBuilder {
	return &mockAsyncOperatorBuilder{}
}

type mockedAsyncOperator struct {
	fDoAsync funcDoAsync
	fFlush   funcFlush
	fClose   funcClose
	fOnError funcOnError
}

func (op mockedAsyncOperator) OnError(ctx context.Context, err error) {
	if op.fOnError != nil {
		op.fOnError(ctx, err)
	}
}

// outWork can be nil
// If `err` is not nil then either `outWork` or `work` (if outWork is nil) is passed to `ICatchOperator`
func (op mockedAsyncOperator) DoAsync(ctx context.Context, work IWorkpiece) (outWork IWorkpiece, err error) {
	if op.fDoAsync != nil {
		return op.fDoAsync(ctx, work)
	}
	return nil, nil
}

func (op mockedAsyncOperator) Flush(callback OpFuncFlush) (err error) {
	if op.fFlush != nil {
		return op.fFlush(callback)
	}
	return nil
}

func (op mockedAsyncOperator) Close() {
	if op.fClose != nil {
		op.fClose()
	}
}
