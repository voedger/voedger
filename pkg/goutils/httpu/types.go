/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package httpu

import (
	"context"
	"io"
	"net/http"
	"time"
)

type implIHTTPClient struct {
	client      *http.Client
	defaultOpts []ReqOptFunc
}

type HTTPResponse struct {
	Body     string
	HTTPResp *http.Response
	Opts     IReqOpts
}

type IReqOpts interface {
	Append(ReqOptFunc)
	ExpectedHTTPCodes() []int
	CustomOpts(key any) (customOpts any)
	httpOpts() *reqOpts
}

type ReqOptFunc func(opts IReqOpts)
type RetryOnStatusOpt func(*retryOnStatus)
type RetryPolicyOpt func(opts IReqOpts)

type IHTTPClient interface {
	Req(ctx context.Context, urlStr string, body string, optFuncs ...ReqOptFunc) (*HTTPResponse, error)
	ReqReader(ctx context.Context, urlStr string, bodyReader io.Reader, optFuncs ...ReqOptFunc) (*HTTPResponse, error)
	CloseIdleConnections()
}

type reqOpts struct {
	method            string
	headers           map[string]string
	cookies           map[string]string
	expectedHTTPCodes []int
	responseHandler   func(httpResp *http.Response) // used if no errors and an expected status code is received
	urlPath           string
	discardResp       bool
	bodyReader        io.Reader
	withoutAuth       bool
	appendedOpts      []ReqOptFunc
	validators        []func(IReqOpts) (panicMessage string)
	retryOnErr        []func(err error) (retry bool)
	retryOnStatus     []retryOnStatus
	customOpts        map[any]any
}

type retryOnStatus struct {
	statusCode        int
	respectRetryAfter bool
	maxRetryDuration  time.Duration
}
