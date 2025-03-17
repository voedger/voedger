/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package bus

import "context"

type IRequestSender interface {
	// err != nil -> responseMeta does not matter, responseCh and responseErr must not be touched
	// responseCh must be read out
	// *responseErr must be checked only after reading out the responseCh
	// caller must eventually close clientCtx
	// ErrSendTimeoutExpired
	SendRequest(clientCtx context.Context, req Request) (responseCh <-chan any, responseMeta ResponseMeta, responseErr *error, err error)
}

type RequestHandler func(requestCtx context.Context, request Request, responder IResponder)

type IResponder interface {
	// panics if called >1 times or after Respond()
	BeginStreamingResponse(statusCode int) IStreamingResponseSenderCloseable

	// panic if called >1 times or after BeginStreamingResponse()
	// obj is string, []byte - text/plain is emitted,
	// obj is error:
	//   complies to interface { ToJSON() string } -> application/json
	//   otherwise -> text/plain
	// otherwise -> application/json
	Respond(statusCode int, obj any) error
}

type IStreamingResponseSenderCloseable interface {
	IStreamingResponseSender
	Close(error)
}

type IStreamingResponseSender interface {
	// ErrNoConsumer
	Send(any) error
}
