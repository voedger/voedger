/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package pipeline

import "context"

type finallyOp[T any] struct {
	NOOP
	f          func(T) error
	finalError error
}

func (o *finallyOp[T]) DoSync(ctx context.Context, work interface{}) (err error) {
	err = o.f(work.(T))
	if o.finalError != nil {
		err := o.finalError
		o.finalError = nil
		return err
	}
	return err
}

func (o *finallyOp[T]) OnErr(err error, _ interface{}, _ IWorkpieceContext) error {
	o.finalError = err
	return nil
}

func FinallyOperator[T any](name string, f func(T) error) *WiredOperator {
	return WireSyncOperator(name, &finallyOp[T]{f: f})
}
