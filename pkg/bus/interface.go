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
	// panics if called >1 times or after BeginCustomResponse()
	BeginApiArrayResponse(statusCode int) IApiArrayResponseWriter

	// panic if called >1 times or after BeginStreamingResponse()
	BeginCustomResponse(meta ResponseMeta) ICustomResponseWriter
}

type IApiArrayResponseWriter interface {
	// Write send item over the bus
	// item -> json.Marshal
	// may may return ErrNoConsumer
	Write(item any) error

	// must be called in the end
	Close(err error)
}

type ICustomResponseWriter interface {
	// Write sends item over the bus
	// item: payload is []byte or string
	// may ErrNoConsumer
	Write(item any) error

	// must be called in the end
	Close()
}
