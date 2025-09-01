/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package coreutils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/goutils/logger"
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
	return func(ro *reqOpts) {
		ro.responseHandler = responseHandler
	}
}

func withBodyReader(bodyReader io.Reader) ReqOptFunc {
	return func(opts *reqOpts) {
		opts.bodyReader = bodyReader
	}
}

// WithLongPolling, WithResponseHandler and WithDiscardResponse are mutual exclusive
func WithLongPolling() ReqOptFunc {
	return func(ro *reqOpts) {
		ro.responseHandler = func(resp *http.Response) {
			if !slices.Contains(ro.expectedHTTPCodes, resp.StatusCode) {
				body, err := readBody(resp)
				if err != nil {
					panic("failed to Read response body in custom response handler: " + err.Error())
				}
				panic(fmt.Sprintf("actual status code %d, expected %v. Body: %s", resp.StatusCode, ro.expectedHTTPCodes, body))
			}
		}
	}
}

// WithDiscardResponse, WithResponseHandler and WithLongPolling are mutual exclusive
// causes FederationReq() to return nil for *HTTPResponse
func WithDiscardResponse() ReqOptFunc {
	return func(opts *reqOpts) {
		opts.discardResp = true
	}
}

func WithoutAuth() ReqOptFunc {
	return func(opts *reqOpts) {
		opts.withoutAuth = true
	}
}

func WithCookies(cookiesPairs ...string) ReqOptFunc {
	return func(po *reqOpts) {
		for i := 0; i < len(cookiesPairs); i += 2 {
			po.cookies[cookiesPairs[i]] = cookiesPairs[i+1]
		}
	}
}

func WithHeaders(headersPairs ...string) ReqOptFunc {
	return func(po *reqOpts) {
		for i := 0; i < len(headersPairs); i += 2 {
			po.headers[headersPairs[i]] = headersPairs[i+1]
		}
	}
}

func WithExpectedCode(expectedHTTPCode int, expectErrorContains ...string) ReqOptFunc {
	return func(po *reqOpts) {
		po.expectedHTTPCodes = append(po.expectedHTTPCodes, expectedHTTPCode)
		po.expectedErrorContains = append(po.expectedErrorContains, expectErrorContains...)
	}
}

// has priority over WithAuthorizeByIfNot
func WithAuthorizeBy(principalToken string) ReqOptFunc {
	return func(po *reqOpts) {
		po.headers[Authorization] = BearerPrefix + principalToken
	}
}

func WithRetryOnCertainError(errMatcher func(err error) bool, timeout time.Duration, retryDelay time.Duration) ReqOptFunc {
	return func(opts *reqOpts) {
		opts.retriersOnErrors = append(opts.retriersOnErrors, retrier{
			macther: errMatcher,
			timeout: timeout,
			delay:   retryDelay,
		})
	}
}

func WithRetryOnAnyError(timeout time.Duration, retryDelay time.Duration) ReqOptFunc {
	return WithRetryOnCertainError(func(error) bool { return true }, timeout, retryDelay)
}

func WithDeadlineOn503(deadline time.Duration) ReqOptFunc {
	return func(opts *reqOpts) {
		opts.deadlineOn503 = deadline
	}
}

func WithRetryOn503() ReqOptFunc {
	return func(opts *reqOpts) {
		opts.skipRetryOn503 = false
	}
}

func WithSkipRetryOn503() ReqOptFunc {
	return func(opts *reqOpts) {
		opts.skipRetryOn503 = true
	}
}

func WithDefaultAuthorize(principalToken string) ReqOptFunc {
	return func(po *reqOpts) {
		if _, ok := po.headers[Authorization]; !ok {
			po.headers[Authorization] = BearerPrefix + principalToken
		}
	}
}

func WithRelativeURL(relativeURL string) ReqOptFunc {
	return func(ro *reqOpts) {
		ro.relativeURL = relativeURL
	}
}

func WithDefaultMethod(m string) ReqOptFunc {
	return func(opts *reqOpts) {
		if len(opts.method) == 0 {
			opts.method = m
		}
	}
}

func WithMethod(m string) ReqOptFunc {
	return func(po *reqOpts) {
		po.method = m
	}
}

func Expect204(expectErrorContains ...string) ReqOptFunc {
	return WithExpectedCode(http.StatusNoContent, expectErrorContains...)
}

func Expect409(expected ...string) ReqOptFunc {
	return WithExpectedCode(http.StatusConflict, expected...)
}

func Expect404(expected ...string) ReqOptFunc {
	return WithExpectedCode(http.StatusNotFound, expected...)
}

func Expect401() ReqOptFunc {
	return WithExpectedCode(http.StatusUnauthorized)
}

func Expect403(expectedMessages ...string) ReqOptFunc {
	return WithExpectedCode(http.StatusForbidden, expectedMessages...)
}

func Expect400(expectErrorContains ...string) ReqOptFunc {
	return WithExpectedCode(http.StatusBadRequest, expectErrorContains...)
}

func Expect405(expectErrorContains ...string) ReqOptFunc {
	return WithExpectedCode(http.StatusMethodNotAllowed, expectErrorContains...)
}

func Expect423(expectErrorContains ...string) ReqOptFunc {
	return WithExpectedCode(http.StatusLocked, expectErrorContains...)
}

func Expect400RefIntegrity_Existence() ReqOptFunc {
	return WithExpectedCode(http.StatusBadRequest, "referential integrity violation", "does not exist")
}

func Expect400RefIntegrity_QName() ReqOptFunc {
	return WithExpectedCode(http.StatusBadRequest, "referential integrity violation", "QNames are only allowed")
}

func Expect429() ReqOptFunc {
	return WithExpectedCode(http.StatusTooManyRequests)
}

func Expect500(expectErrorContains ...string) ReqOptFunc {
	return WithExpectedCode(http.StatusInternalServerError, expectErrorContains...)
}

func Expect503() ReqOptFunc {
	return WithExpectedCode(http.StatusServiceUnavailable)
}

func Expect410() ReqOptFunc {
	return WithExpectedCode(http.StatusGone)
}

func WithOptsValidator(validator func(*reqOpts) (panicMessage string)) ReqOptFunc {
	return func(opts *reqOpts) {
		opts.validators = append(opts.validators, validator)
	}
}

type reqOpts struct {
	method                string
	headers               map[string]string
	cookies               map[string]string
	expectedHTTPCodes     []int
	expectedErrorContains []string
	responseHandler       func(httpResp *http.Response) // used if no errors and an expected status code is received
	relativeURL           string
	discardResp           bool
	retriersOnErrors      []retrier
	bodyReader            io.Reader
	withoutAuth           bool
	skipRetryOn503        bool
	deadlineOn503         time.Duration
	validators            []func(*reqOpts) (panicMessage string)
}

// body and bodyReader are mutual exclusive
func req(method, url, body string, bodyReader io.Reader, headers, cookies map[string]string) (req *http.Request, err error) {
	if bodyReader != nil {
		req, err = http.NewRequest(method, url, bodyReader)
	} else {
		req, err = http.NewRequest(method, url, bytes.NewReader([]byte(body)))
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

func mutualExclusiveOptsValidator(opts *reqOpts) (panicMessage string) {
	mutualExclusiveOpts := 0
	if opts.discardResp {
		mutualExclusiveOpts++
	}
	if opts.responseHandler != nil {
		mutualExclusiveOpts++
	}
	if mutualExclusiveOpts > 1 {
		return "request options conflict"
	}
	return ""
}

func (c *implIHTTPClient) req(ctx context.Context, urlStr string, body string, optFuncs ...ReqOptFunc) (*HTTPResponse, error) {
	opts := &reqOpts{
		headers: map[string]string{},
		cookies: map[string]string{},
		validators: []func(*reqOpts) (panicMessage string){
			mutualExclusiveOptsValidator,
		},
	}
	optFuncs = append(optFuncs, WithRetryOnCertainError(func(err error) bool {
		// https://github.com/voedger/voedger/issues/1694
		return IsWSAEError(err, WSAECONNREFUSED)
	}, retryOn_WSAECONNREFUSED_Timeout, retryOn_WSAECONNREFUSED_Delay))
	for _, defaultOptFunc := range c.defaultOps {
		defaultOptFunc(opts)
	}
	for _, optFunc := range optFuncs {
		optFunc(opts)
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
	var resp *http.Response
	var err error
	tryNum := 0
	startTime := time.Now()

	reqCtx, cancel := context.WithTimeout(ctx, maxHTTPRequestTimeout)
	defer cancel()
reqLoop:
	for reqCtx.Err() == nil {
		req, err := req(opts.method, urlStr, body, opts.bodyReader, opts.headers, opts.cookies)
		if err != nil {
			return nil, err
		}
		resp, err = c.client.Do(req)
		if err != nil {
			for _, retrier := range opts.retriersOnErrors {
				if retrier.macther(err) {
					if time.Since(startTime) < retrier.timeout {
						time.Sleep(retrier.delay)
						continue reqLoop
					}
				}
			}
			return nil, fmt.Errorf("request do() failed: %w", err)
		}
		if opts.responseHandler == nil {
			defer resp.Body.Close()
		}
		if resp.StatusCode == http.StatusServiceUnavailable && !slices.Contains(opts.expectedHTTPCodes, http.StatusServiceUnavailable) &&
			!opts.skipRetryOn503 {
			if opts.deadlineOn503 > 0 && time.Since(startTime) > opts.deadlineOn503 {
				break
			}
			if err := discardRespBody(resp); err != nil {
				return nil, err
			}
			logger.Verbose("503. retrying...")
			if tryNum > shortRetriesOn503Amount {
				time.Sleep(longRetryOn503Delay)
			} else {
				time.Sleep(shortRetryOn503Delay)
			}
			tryNum++
			continue
		}
		break
	}
	if reqCtx.Err() != nil {
		return nil, reqCtx.Err()
	}
	isCodeExpected := slices.Contains(opts.expectedHTTPCodes, resp.StatusCode)
	if isCodeExpected && opts.discardResp {
		err := discardRespBody(resp)
		return nil, err
	}
	httpResponse := &HTTPResponse{
		HTTPResp:          resp,
		expectedHTTPCodes: opts.expectedHTTPCodes,
	}
	if resp.StatusCode == http.StatusOK && isCodeExpected && opts.responseHandler != nil {
		opts.responseHandler(resp)
		return httpResponse, nil
	}
	respBody, err := readBody(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	httpResponse.Body = respBody
	var statusErr error
	if !isCodeExpected {
		statusErr = fmt.Errorf("%w: %d, %s", ErrUnexpectedStatusCode, resp.StatusCode, respBody)
	}
	return httpResponse, statusErr
}

func (c *implIHTTPClient) CloseIdleConnections() {
	c.client.CloseIdleConnections()
}

func (resp *HTTPResponse) ExpectedHTTPCodes() []int {
	return resp.expectedHTTPCodes
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

func (resp *HTTPResponse) getError(t *testing.T) map[string]interface{} {
	t.Helper()
	m := map[string]interface{}{}
	err := json.Unmarshal([]byte(resp.Body), &m)
	require.NoError(t, err)
	return m["sys.Error"].(map[string]interface{})
}

func (resp *HTTPResponse) RequireError(t *testing.T, message string) {
	t.Helper()
	m := resp.getError(t)
	require.Equal(t, message, m["Message"])
}

func (resp *HTTPResponse) RequireContainsError(t *testing.T, messagePart string) {
	t.Helper()
	m := resp.getError(t)
	require.Contains(t, m["Message"], messagePart)
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
	client     *http.Client
	defaultOps []ReqOptFunc
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
		client:     &http.Client{Transport: tr},
		defaultOps: defaultOpts,
	}
	return client, client.CloseIdleConnections
}

func DenyGETAndDiscardResponse(opts *reqOpts) (panicMessage string) {
	if opts.discardResp && opts.method == http.MethodGet {
		return "WithDiscardResponse is denied on GET method"
	}
	return ""
}
