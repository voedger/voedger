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
	InitResponse(responseMeta ResponseMeta) IResponseWriter

	// panics if called >1 times or after InitResponse
	Respond(statusCode int, obj any) error

	// need to distinguish cmd normal reply with SysError in body from cmd error
	// RespondSysError(sysError coreutils.SysError) error
}

type IResponseWriter interface {
	Write(obj any) error
	Close(err error)
}
