/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package ibus

import (
	"context"
	"time"
)

type CLIParams struct {
	MaxNumOfConcurrentRequests int
	ReadWriteTimeout           time.Duration
}

type IBus interface {
	// panics if receivers already exists
	// NOTE: EchoReceiver can be used for testing purposes
	RegisterReceiver(owner string, app string, partition int, part string, r Receiver, numOfProcessors int, bufferSize int)

	// ok is false if receivers were not found
	UnregisterReceiver(owner string, app string, partition int, part string) (ok bool)

	// If appropriate receivers do not exist then "BadRequestSender" should be returned
	// "BadRequestSender" returns an StatusBadRequest(400) error for every request
	QuerySender(owner string, app string, partition int, part string) (sender ISender, ok bool)

	GetMetrics() (metrics Metrics)
}

type Metrics struct {
	MaxNumOfConcurrentRequests int
	NumOfConcurrentRequests    int
}

type SectionsHandlerType func(section interface{})

type SectionsWriterType interface {
	// Result is false if client cancels the request or receiver is being unregistered
	Write(section interface{}) bool
}

type ISender interface {
	// err.Error() must have QName format:
	//   var ErrBusTimeoutExpired = errors.New("coreutils.ErrSendTimeoutExpired")
	// NullHandler can be used as a reader
	Send(ctx context.Context, request interface{}, sectionsHandler SectionsHandlerType) (response interface{}, status Status, err error)
}

// Receiver must check context
// err.Error() must have QName format
type Receiver func(processorsCtx context.Context, request interface{}, sectionsWriter SectionsWriterType) (response interface{}, status Status, err error)

type Status struct {
	// Ref. https://go.dev/src/net/http/status.go
	// StatusBadRequest(400) if server got the request but could not process it
	// StatusGatewayTimeout(504) if timeout expired
	HTTPStatus   int
	ErrorMessage string
	ErrorData    string
}
