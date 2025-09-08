/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package coreutils

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/voedger/voedger/pkg/goutils/logger"
	retrier "github.com/voedger/voedger/pkg/goutils/retry"
	"github.com/voedger/voedger/pkg/istructs"
	"golang.org/x/exp/slices"
)

func NewHTTPErrorf(httpStatus int, args ...interface{}) SysError {
	return SysError{
		HTTPStatus: httpStatus,
		Message:    fmt.Sprint(args...),
	}
}

func NewHTTPError(httpStatus int, err error) SysError {
	return NewHTTPErrorf(httpStatus, err.Error())
}

// WithResponseHandler, WithLongPolling and WithDiscardResponse are mutual exclusive
func WithResponseHandler(responseHandler func(httpResp *http.Response)) ReqOptFunc {
	return func(opts IReqOpts) {
		opts.httpOpts().responseHandler = responseHandler
	}
}

func withBodyReader(bodyReader io.Reader) ReqOptFunc {
	return func(opts IReqOpts) {
		opts.httpOpts().bodyReader = bodyReader
	}
}

// WithLongPolling, WithResponseHandler and WithDiscardResponse are mutual exclusive
func WithLongPolling() ReqOptFunc {
	return func(opts IReqOpts) {
		opts.httpOpts().responseHandler = func(resp *http.Response) {
			if !slices.Contains(opts.httpOpts().expectedHTTPCodes, resp.StatusCode) {
				body, err := readBody(resp)
				if err != nil {
					panic("failed to Read response body in custom response handler: " + err.Error())
				}
				panic(fmt.Sprintf("actual status code %d, expected %v. Body: %s", resp.StatusCode, opts.httpOpts().expectedHTTPCodes, body))
			}
		}
	}
}

// WithDiscardResponse, WithResponseHandler and WithLongPolling are mutual exclusive
// causes FederationReq() to return nil for *HTTPResponse
func WithDiscardResponse() ReqOptFunc {
	return func(opts IReqOpts) {
		opts.httpOpts().discardResp = true
	}
}

func WithoutAuth() ReqOptFunc {
	return func(opts IReqOpts) {
		opts.httpOpts().withoutAuth = true
	}
}

func WithCookies(cookiesPairs ...string) ReqOptFunc {
	return func(opts IReqOpts) {
		for i := 0; i < len(cookiesPairs); i += 2 {
			opts.httpOpts().cookies[cookiesPairs[i]] = cookiesPairs[i+1]
		}
	}
}

func WithHeaders(headersPairs ...string) ReqOptFunc {
	return func(opts IReqOpts) {
		for i := 0; i < len(headersPairs); i += 2 {
			opts.httpOpts().headers[headersPairs[i]] = headersPairs[i+1]
		}
	}
}

func WithExpectedCode(expectedHTTPCode int) ReqOptFunc {
	return func(opts IReqOpts) {
		opts.httpOpts().expectedHTTPCodes = append(opts.httpOpts().expectedHTTPCodes, expectedHTTPCode)
	}
}

// has priority over WithAuthorizeByIfNot
func WithAuthorizeBy(principalToken string) ReqOptFunc {
	return func(opts IReqOpts) {
		opts.httpOpts().headers[Authorization] = BearerPrefix + principalToken
	}
}

func WithMaxRetryDurationOn503(maxRetryDuration time.Duration) ReqOptFunc {
	return func(opts IReqOpts) {
		opts.httpOpts().maxRetryDurationOn503 = maxRetryDuration
	}
}

func WithRetryOn503() ReqOptFunc {
	return func(opts IReqOpts) {
		opts.httpOpts().skipRetryOn503 = false
	}
}

func WithSkipRetryOn503() ReqOptFunc {
	return func(opts IReqOpts) {
		opts.httpOpts().skipRetryOn503 = true
	}
}

func WithDefaultAuthorize(principalToken string) ReqOptFunc {
	return func(opts IReqOpts) {
		if _, ok := opts.httpOpts().headers[Authorization]; !ok {
			opts.httpOpts().headers[Authorization] = BearerPrefix + principalToken
		}
	}
}

func WithRelativeURL(relativeURL string) ReqOptFunc {
	return func(opts IReqOpts) {
		opts.httpOpts().relativeURL = relativeURL
	}
}

func WithDefaultMethod(m string) ReqOptFunc {
	return func(opts IReqOpts) {
		if len(opts.httpOpts().method) == 0 {
			opts.httpOpts().method = m
		}
	}
}

func WithMethod(m string) ReqOptFunc {
	return func(opts IReqOpts) {
		opts.httpOpts().method = m
	}
}

func WithRetryErrorMatcher(matcher func(err error) (retry bool)) ReqOptFunc {
	return func(opts IReqOpts) {
		opts.httpOpts().retryErrsMatchers = append(opts.httpOpts().retryErrsMatchers, matcher)
	}
}

func WithCustomOptsProvider(prov func(internalOpts IReqOpts) (customOpts IReqOpts)) ReqOptFunc {
	return func(opts IReqOpts) {
		opts.httpOpts().customOptsProvider = prov
	}
}

func Expect204() ReqOptFunc {
	return WithExpectedCode(http.StatusNoContent)
}

func Expect409() ReqOptFunc {
	return WithExpectedCode(http.StatusConflict)
}

func Expect404() ReqOptFunc {
	return WithExpectedCode(http.StatusNotFound)
}

func Expect401() ReqOptFunc {
	return WithExpectedCode(http.StatusUnauthorized)
}

func Expect403() ReqOptFunc {
	return WithExpectedCode(http.StatusForbidden)
}

func Expect400() ReqOptFunc {
	return WithExpectedCode(http.StatusBadRequest)
}

func Expect405() ReqOptFunc {
	return WithExpectedCode(http.StatusMethodNotAllowed)
}

func Expect423() ReqOptFunc {
	return WithExpectedCode(http.StatusLocked)
}

func Expect429() ReqOptFunc {
	return WithExpectedCode(http.StatusTooManyRequests)
}

func Expect500() ReqOptFunc {
	return WithExpectedCode(http.StatusInternalServerError)
}

func Expect503() ReqOptFunc {
	return WithExpectedCode(http.StatusServiceUnavailable)
}

func Expect410() ReqOptFunc {
	return WithExpectedCode(http.StatusGone)
}

func WithOptsValidator(validator func(IReqOpts) (panicMessage string)) ReqOptFunc {
	return func(opts IReqOpts) {
		opts.httpOpts().validators = append(opts.httpOpts().validators, validator)
	}
}

type reqOpts struct {
	method                string
	headers               map[string]string
	cookies               map[string]string
	expectedHTTPCodes     []int
	responseHandler       func(httpResp *http.Response) // used if no errors and an expected status code is received
	relativeURL           string
	discardResp           bool
	bodyReader            io.Reader
	withoutAuth           bool
	skipRetryOn503        bool
	maxRetryDurationOn503 time.Duration
	customOptsProvider    func(IReqOpts) IReqOpts
	appendedOpts          []ReqOptFunc
	validators            []func(IReqOpts) (panicMessage string)
	retryErrsMatchers     []func(err error) (retry bool)
}

// body and bodyReader are mutual exclusive
func req(ctx context.Context, method, url, body string, bodyReader io.Reader, headers, cookies map[string]string) (req *http.Request, err error) {
	if bodyReader != nil {
		req, err = http.NewRequestWithContext(ctx, method, url, bodyReader)
	} else {
		req, err = http.NewRequestWithContext(ctx, method, url, bytes.NewReader([]byte(body)))
	}
	if err != nil {
		return nil, fmt.Errorf("NewRequest() failed: %w", err)
	}
	req.Close = true
	for k, v := range headers {
		req.Header.Add(k, v)
	}
	for k, v := range cookies {
		req.AddCookie(&http.Cookie{
			Name:  k,
			Value: v,
		})
	}
	return req, nil
}

func (c *implIHTTPClient) ReqReader(ctx context.Context, urlStr string, bodyReader io.Reader, optFuncs ...ReqOptFunc) (*HTTPResponse, error) {
	optFuncs = append(optFuncs, withBodyReader(bodyReader))
	return c.req(ctx, urlStr, "", optFuncs...)
}

// status code expected -> DiscardBody, ResponseHandler are used
// status code is unexpected -> DiscardBody, ResponseHandler are ignored, body is read out, wrapped ErrUnexpectedStatusCode is returned
func (c *implIHTTPClient) Req(ctx context.Context, urlStr string, body string, optFuncs ...ReqOptFunc) (*HTTPResponse, error) {
	return c.req(ctx, urlStr, body, optFuncs...)
}

func optsValidator_responseHandling(opts IReqOpts) (panicMessage string) {
	mutualExclusiveOpts := 0
	o := opts.httpOpts()
	if o.discardResp {
		mutualExclusiveOpts++
	}
	if o.responseHandler != nil {
		mutualExclusiveOpts++
	}
	if mutualExclusiveOpts > 1 {
		return "request options conflict"
	}
	return ""
}

func (opts *reqOpts) Append(opt ReqOptFunc) {
	opts.appendedOpts = append(opts.appendedOpts, opt)
}

func (opts *reqOpts) ExpectedHTTPCodes() []int {
	return opts.expectedHTTPCodes
}

func (opts *reqOpts) httpOpts() *reqOpts {
	return opts
}

func optsValidator_retryOn503(opts IReqOpts) (panicMessage string) {
	if opts.httpOpts().maxRetryDurationOn503 > 0 && opts.httpOpts().skipRetryOn503 {
		return "max retry duration on 503 cannot be specified if skip on 503 is set"
	}
	return ""
}

func (c *implIHTTPClient) req(ctx context.Context, urlStr string, body string, optFuncs ...ReqOptFunc) (*HTTPResponse, error) {
	opts := &reqOpts{
		headers: map[string]string{},
		cookies: map[string]string{},
		validators: []func(IReqOpts) (panicMessage string){
			optsValidator_responseHandling,
			optsValidator_retryOn503,
		},
	}
	for _, defaultOptFunc := range c.defaultOpts {
		defaultOptFunc(opts)
	}
	var iOpts IReqOpts = opts
	var prevCustomOptsProvider func(IReqOpts) IReqOpts = nil
	for _, optFunc := range optFuncs {
		optFunc(iOpts)
		if prevCustomOptsProvider == nil && opts.customOptsProvider != nil {
			iOpts = opts.customOptsProvider(iOpts)
			prevCustomOptsProvider = opts.customOptsProvider
		}
	}
	for _, optFunc := range opts.appendedOpts {
		optFunc(iOpts)
	}
	if len(opts.method) == 0 {
		opts.method = http.MethodGet
	}

	if len(opts.expectedHTTPCodes) == 0 {
		opts.expectedHTTPCodes = append(opts.expectedHTTPCodes, http.StatusOK, http.StatusCreated)
	}
	if len(opts.relativeURL) > 0 {
		netURL, err := url.Parse(urlStr)
		if err != nil {
			return nil, err
		}
		netURL.Path = opts.relativeURL
		urlStr = netURL.String()
	}
	if opts.withoutAuth {
		delete(opts.headers, Authorization)
		delete(opts.cookies, Authorization)
	}

	for _, v := range opts.validators {
		if panicMessage := v(opts); len(panicMessage) > 0 {
			panic(panicMessage)
		}
	}
	startTime := time.Now()

	reqCtx, cancel := context.WithTimeout(ctx, maxHTTPRequestTimeout)
	defer cancel()

	retrierCfg := retrier.NewConfig(httpBaseRetryDelay, httpMaxRetryDelay)
	retrierCfg.OnError = func(attempt int, delay time.Duration, opErr error) (retry bool, abortErr error) {
		for _, matcher := range opts.retryErrsMatchers {
			if matcher(opErr) {
				return true, nil
			}
		}
		return false, fmt.Errorf("request failed: %w", opErr)
	}

	resp, err := retrier.Retry(reqCtx, retrierCfg, func() (*http.Response, error) {
		req, err := req(ctx, opts.method, urlStr, body, opts.bodyReader, opts.headers, opts.cookies)
		if err != nil {
			return nil, err
		}
		resp, err := c.client.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode == http.StatusServiceUnavailable && opts.shouldHandle503() {
			if opts.maxRetryDurationOn503 > 0 && time.Since(startTime) > opts.maxRetryDurationOn503 {
				return resp, nil
			}
			defer resp.Body.Close()
			if err := discardRespBody(resp); err != nil {
				return nil, err
			}
			logger.Verbose("503. retrying...")
			return nil, errHTTPStatus503
		}
		return resp, nil
	})
	if err != nil {
		return nil, err
	}
	isCodeExpected := slices.Contains(opts.expectedHTTPCodes, resp.StatusCode)
	if isCodeExpected && opts.discardResp {
		err := discardRespBody(resp)
		return nil, err
	}
	httpResponse := &HTTPResponse{
		HTTPResp: resp,
		Opts:     iOpts,
	}
	if resp.StatusCode == http.StatusOK && isCodeExpected && opts.responseHandler != nil {
		opts.responseHandler(resp)
		return httpResponse, nil
	}
	httpResponse.Body, err = readBody(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	var statusErr error
	if !isCodeExpected {
		statusErr = fmt.Errorf("%w: %d, %s", ErrUnexpectedStatusCode, resp.StatusCode, httpResponse.Body)
	}
	return httpResponse, statusErr
}

func (c *implIHTTPClient) CloseIdleConnections() {
	c.client.CloseIdleConnections()
}

func (resp *HTTPResponse) Println() {
	log.Println(resp.Body)
}

func (resp *HTTPResponse) PrintJSON() {
	obj := make(map[string]interface{})
	err := json.Unmarshal([]byte(resp.Body), &obj)
	if err != nil {
		log.Fatalln(err)
	}
	bb, err := json.MarshalIndent(obj, "", "	")
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("\n", string(bb))
}

func readBody(resp *http.Response) (string, error) {
	respBody, err := io.ReadAll(resp.Body)
	return string(respBody), err
}

func discardRespBody(resp *http.Response) error {
	_, err := io.Copy(io.Discard, resp.Body)
	if err != nil {
		// https://github.com/voedger/voedger/issues/1694
		if !IsWSAEError(err, WSAECONNRESET) {
			return fmt.Errorf("failed to discard response body: %w", err)
		}
	}
	return nil
}

func (resp *FuncResponse) Len() int {
	return resp.NumRows()
}

func (resp *FuncResponse) NumRows() int {
	if resp.IsEmpty() {
		return 0
	}
	return len(resp.Sections[0].Elements)
}

func (resp *FuncResponse) SectionRow(rowIdx ...int) []interface{} {
	if len(rowIdx) > 1 {
		panic("must be 0 or 1 rowIdx'es")
	}
	if len(resp.Sections) == 0 {
		panic("empty response")
	}
	i := 0
	if len(rowIdx) == 1 {
		i = rowIdx[0]
	}
	return resp.Sections[0].Elements[i][0][0]
}

// returns a new ID for raw ID 1
func (resp *FuncResponse) NewID() istructs.RecordID {
	return resp.NewIDs["1"]
}

func (resp *FuncResponse) IsEmpty() bool {
	return len(resp.Sections) == 0 && len(resp.QPv2Response) == 0
}

type implIHTTPClient struct {
	client      *http.Client
	defaultOpts []ReqOptFunc
}

var constDefaultOpts = []ReqOptFunc{
	WithRetryErrorMatcher(func(err error) bool {
		// https://github.com/voedger/voedger/issues/1694
		return IsWSAEError(err, WSAECONNREFUSED)
	}),
	WithRetryErrorMatcher(func(err error) bool {
		// retry on 503
		return errors.Is(err, errHTTPStatus503)
	}),
}

func NewIHTTPClient(defaultOpts ...ReqOptFunc) (client IHTTPClient, clenup func()) {
	// set linger - see https://github.com/voedger/voedger/issues/415
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		dialer := net.Dialer{}
		conn, err := dialer.DialContext(ctx, network, addr)
		if err != nil {
			return nil, err
		}

		err = conn.(*net.TCPConn).SetLinger(0)
		return conn, err
	}
	client = &implIHTTPClient{
		client:      &http.Client{Transport: tr},
		defaultOpts: append(slices.Clone(constDefaultOpts), defaultOpts...),
	}
	return client, client.CloseIdleConnections
}

func DenyGETAndDiscardResponse(opts IReqOpts) (panicMessage string) {
	if opts.httpOpts().discardResp && opts.httpOpts().method == http.MethodGet {
		return "WithDiscardResponse is denied on GET method"
	}
	return ""
}

func (o reqOpts) shouldHandle503() bool {
	return !slices.Contains(o.expectedHTTPCodes, http.StatusServiceUnavailable) && !o.skipRetryOn503
}
