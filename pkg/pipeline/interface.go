// Copyright (c) 2021-present Voedger Authors.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.
// @author Maxim Geraskin

package pipeline

import (
	"context"
)

type StorageID string

type IPipeline interface {
	Close()
}

type OpFuncFlush func(work IWorkpiece)

type IOperator interface {
	// Must be the last call to operator
	Close()
}

type ISyncOperator interface {
	IOperator
	// If `err` is not nil then `work` is passed to the nearest `ICatch`
	DoSync(ctx context.Context, work IWorkpiece) (err error)
}

type IAsyncOperator interface {
	IOperator

	// outWork can be nil
	// If `err` is not nil then either `outWork` or `work` (if outWork is nil) is passed through the channel together with `err`
	DoAsync(ctx context.Context, work IWorkpiece) (outWork IWorkpiece, err error)

	// callback may be called few times
	// Close() may be called in the middle of flush
	Flush(callback OpFuncFlush) (err error)

	// Called when the error has happened in one of previous operators.
	// Called once per operator, then the error is forwarded to the next operator
	OnError(ctx context.Context, err error)
}

type IAsyncPipeline interface {
	IPipeline
	SendAsync(work IWorkpiece) (err error)
}

type ISyncPipeline interface {
	IPipeline
	ISyncOperator
	SendSync(work IWorkpiece) (err error)
}

type IWorkpiece interface {
	Release()
}

type IWorkpieceContext interface {
	GetPipelineName() string
	GetPipelineStruct() string
}

// ********  Operators logic

type ICatch interface {
	// OnErr handles the error in sync operator.
	// If newErr is returned and the original err may further be needed, the developer should wrap the original error into returned newErr.
	OnErr(err error, work interface{}, context IWorkpieceContext) (newErr error)
}

type ISwitch interface {
	Switch(work interface{}) (branchName string, err error)
}

type Fork func(work IWorkpiece, branchNumber int) (fork IWorkpiece, err error)

func ForkSame(work IWorkpiece, _ int) (fork IWorkpiece, err error) { return work, nil }

type IService interface {
	// Prepare tunes-up the service: checks arguments from `work` etc
	// if error is nil then Run() has no right to fail to start
	// e.g. if the service has to work with some outer system then connection should be established in Prepare() and communication cycle should be started in Run()
	Prepare(work interface{}) error

	// Run could be blocking or non-blocking
	Run(ctx context.Context)

	// ctx provided in Run() should be cancelled right before Stop()
	Stop()
}

type IServiceEx interface {
	IService
	// Blocking RunEx implementation must call started()
	RunEx(ctx context.Context, started func())
}
