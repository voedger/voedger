/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package coreutils

import (
	"context"

	"github.com/voedger/voedger/pkg/pipeline"
)

type deferOp[T any] struct {
	pipeline.NOOP
	f          func(T) error
	deferError error
}

func (o *deferOp[T]) DoSync(ctx context.Context, work interface{}) (err error) {
	err = o.f(work.(T))
	if o.deferError != nil {
		err := o.deferError
		o.deferError = nil
		return err
	}
	return err
}

func (o *deferOp[T]) OnErr(err error, _ T, _ pipeline.IWorkpieceContext) error {
	o.deferError = err
	return nil
}

func DeferOperator[T any](name string, f func(T) error) *pipeline.WiredOperator {
	return pipeline.WireSyncOperator(name, &deferOp[T]{f: f})
}
