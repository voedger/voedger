/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package httpu

import (
	"errors"
	"net"
	"syscall"
	"time"
)

const (
	Authorization                                = "Authorization"
	ContentType                                  = "Content-Type"
	ContentDisposition                           = "Content-Disposition"
	Accept                                       = "Accept"
	Origin                                       = "Origin"
	ContentType_ApplicationJSON                  = "application/json"
	ContentType_ApplicationXBinary               = "application/x-binary"
	ContentType_TextPlain                        = "text/plain"
	ContentType_TextHTML                         = "text/html"
	ContentType_MultipartFormData                = "multipart/form-data"
	BearerPrefix                                 = "Bearer "
	WSAECONNRESET                  syscall.Errno = 10054
	WSAECONNREFUSED                syscall.Errno = 10061
	maxHTTPRequestTimeout                        = time.Hour
	httpBaseRetryDelay                           = 20 * time.Millisecond
	httpMaxRetryDelay                            = 1 * time.Second
	localhostDynamic                             = "127.0.0.1:0"
)

var (
	constDefaultOpts = []ReqOptFunc{
		WithRetryErrorMatcher(func(err error) bool {
			// https://github.com/voedger/voedger/issues/1694
			return IsWSAEError(err, WSAECONNREFUSED)
		}),
		WithRetryErrorMatcher(func(err error) bool {
			// retry on 503
			return errors.Is(err, errHTTPStatus503)
		}),
	}
	LocalhostIP = net.IPv4(127, 0, 0, 1)
)
