/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package pipeline

import (
	"context"
)

type serviceOperator[T any] struct {
	serviceDone chan struct{}
	iService    IService
	isStarted   bool
}

func (so *serviceOperator[T]) Close() {
	if !so.isStarted {
		return
	}
	so.iService.Stop()
	<-so.serviceDone
}

func (so *serviceOperator[T]) DoSync(ctx context.Context, work T) (err error) {
	if err = so.iService.Prepare(work); err != nil {
		return err
	}
	so.isStarted = true
	so.serviceDone = make(chan struct{})
	go func() {
		so.iService.Run(ctx)
		close(so.serviceDone)
	}()
	return
}

func ServiceOperator[T any](service IService) ISyncOperator[T] {
	if service == nil {
		panic("service logic must not be nil")
	}
	return &serviceOperator[T]{
		iService: service,
	}
}
