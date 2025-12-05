/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package httpu

import (
	"errors"
	"net"
	"net/http"
	"syscall"
	"time"
)

const (
	Authorization                                = "Authorization"
	ContentType                                  = "Content-Type"
	ContentDisposition                           = "Content-Disposition"
	Accept                                       = "Accept"
	Origin                                       = "Origin"
	RetryAfter                                   = "Retry-After"
	ContentType_ApplicationJSON                  = "application/json"
	ContentType_ApplicationXBinary               = "application/x-binary"
	ContentType_TextPlain                        = "text/plain"
	ContentType_TextHTML                         = "text/html"
	ContentType_TextEventStream                  = "text/event-stream"
	ContentType_MultipartFormData                = "multipart/form-data"
	BearerPrefix                                 = "Bearer "
	WSAECONNRESET                  syscall.Errno = 10054
	maxHTTPRequestTimeout                        = time.Hour
	httpBaseRetryDelay                           = 20 * time.Millisecond
	httpMaxRetryDelay                            = 1 * time.Second
	localhostDynamic                             = "127.0.0.1:0"
	defaultMaxRetryDuration                      = time.Minute
)

var (
	mandatoryOpts = []ReqOptFunc{
		ReqOptFunc(WithRetryOnError(func(err error) bool {
			return errors.Is(err, errRetry)
		})),
	}
	DefaultRetryPolicyOpts = []RetryPolicyOpt{
		WithRetryOnStatus(http.StatusRequestTimeout),
		WithRetryOnStatus(http.StatusTooManyRequests, WithRespectRetryAfter()),
		WithRetryOnStatus(http.StatusInternalServerError),
		WithRetryOnStatus(http.StatusBadGateway),
		WithRetryOnStatus(http.StatusServiceUnavailable),
		WithRetryOnStatus(http.StatusGatewayTimeout),
	}
	LocalhostIP = net.IPv4(127, 0, 0, 1)
)
