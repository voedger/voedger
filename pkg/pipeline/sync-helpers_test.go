// Copyright (c) 2021-present Voedger Authors.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.
// @author Michael Saigachenko

package pipeline

import "context"

type funcDoSync func(ctx context.Context, work interface{}) (err error)
type funcClose func()
type funcOnError func(ctx context.Context, err error)

type mockSyncOperatorBuilder struct {
	fDoSync funcDoSync
	fOnErr  funcOnErr
	fClose  funcClose
}

func (b *mockSyncOperatorBuilder) doSync(f funcDoSync) *mockSyncOperatorBuilder {
	b.fDoSync = f
	return b
}
func (b *mockSyncOperatorBuilder) catch(f funcOnErr) *mockSyncOperatorBuilder {
	b.fOnErr = f
	return b
}
func (b *mockSyncOperatorBuilder) close(f funcClose) *mockSyncOperatorBuilder {
	b.fClose = f
	return b
}
func (b *mockSyncOperatorBuilder) create() ISyncOperator {
	if b.fOnErr != nil {
		return &mockedSyncCatchOperator{
			fOnErr:  b.fOnErr,
			fDoSync: b.fDoSync,
		}
	}
	return &mockedSyncOperator{
		fDoSync: b.fDoSync,
		fClose:  b.fClose,
	}
}
func mockSyncOp() *mockSyncOperatorBuilder {
	return &mockSyncOperatorBuilder{}
}

type mockedSyncOperator struct {
	fDoSync funcDoSync
	fClose  funcClose
}

// outWork can be nil
// If `err` is not nil then either `outWork` or `work` (if outWork is nil) is passed to `ICatchOperator`
func (op mockedSyncOperator) DoSync(ctx context.Context, work IWorkpiece) (err error) {
	if op.fDoSync != nil {
		return op.fDoSync(ctx, work)
	}
	return nil
}

func (op mockedSyncOperator) Close() {
	if op.fClose != nil {
		op.fClose()
	}
}

type mockedSyncCatchOperator struct {
	fOnErr  funcOnErr
	fDoSync funcDoSync
}

func (o *mockedSyncCatchOperator) OnErr(err error, work interface{}, context IWorkpieceContext) (newErr error) {
	if o.fOnErr != nil {
		return o.fOnErr(err, work, context)
	}
	return nil
}
func (o mockedSyncCatchOperator) DoSync(ctx context.Context, work IWorkpiece) (err error) {
	if o.fDoSync != nil {
		return o.fDoSync(ctx, work)
	}
	return nil
}

func (o mockedSyncCatchOperator) Close() {

}
