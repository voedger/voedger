/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package bus

import (
	"context"
)

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
	// panics if called >1 times or after Respond
	// ContentType is ApplicationJSON
	InitResponse(statusCode int) IResponseWriter

	// panics if called >1 times or after InitResponse
	Respond(responseMeta ResponseMeta, obj any) error
}

type IResponseWriter interface {
	Write(obj any) error
	Close(err error)
}
