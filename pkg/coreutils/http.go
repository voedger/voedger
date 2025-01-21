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
	"strings"
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

func WithAuthorizeByIfNot(principalToken string) ReqOptFunc {
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

func WithMethod(m string) ReqOptFunc {
	return func(po *reqOpts) {
		po.method = m
	}
}

func Expect409(expected ...string) ReqOptFunc {
	return WithExpectedCode(http.StatusConflict, expected...)
}

func Expect404() ReqOptFunc {
	return WithExpectedCode(http.StatusNotFound)
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

func ExpectSysError500() ReqOptFunc {
	return func(opts *reqOpts) {
		opts.expectedSysErrorCode = http.StatusInternalServerError
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
	expectedSysErrorCode  int
	retriersOnErrors      []retrier
	bodyReader            io.Reader
	withoutAuth           bool
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

func (c *implIHTTPClient) ReqReader(urlStr string, bodyReader io.Reader, optFuncs ...ReqOptFunc) (*HTTPResponse, error) {
	optFuncs = append(optFuncs, withBodyReader(bodyReader))
	return c.req(urlStr, "", optFuncs...)
}

// status code expected -> DiscardBody, ResponseHandler are used
// status code is unexpected -> DiscardBody, ResponseHandler are ignored, body is read out, wrapped ErrUnexpectedStatusCode is returned
func (c *implIHTTPClient) Req(urlStr string, body string, optFuncs ...ReqOptFunc) (*HTTPResponse, error) {
	return c.req(urlStr, body, optFuncs...)
}

func (c *implIHTTPClient) req(urlStr string, body string, optFuncs ...ReqOptFunc) (*HTTPResponse, error) {
	opts := &reqOpts{
		headers: map[string]string{},
		cookies: map[string]string{},
		method:  http.MethodGet,
	}
	optFuncs = append(optFuncs, WithRetryOnCertainError(func(err error) bool {
		// https://github.com/voedger/voedger/issues/1694
		return IsWSAEError(err, WSAECONNREFUSED)
	}, retryOn_WSAECONNREFUSED_Timeout, retryOn_WSAECONNREFUSED_Delay))
	for _, optFunc := range optFuncs {
		optFunc(opts)
	}

	mutualExclusiveOpts := 0
	if opts.discardResp {
		mutualExclusiveOpts++
	}
	if opts.expectedSysErrorCode > 0 {
		mutualExclusiveOpts++
	}
	if opts.responseHandler != nil {
		mutualExclusiveOpts++
	}
	if mutualExclusiveOpts > 1 {
		panic("request options conflict")
	}

	if len(opts.expectedHTTPCodes) == 0 {
		opts.expectedHTTPCodes = append(opts.expectedHTTPCodes, http.StatusOK)
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
	var resp *http.Response
	var err error
	tryNum := 0
	startTime := time.Now()

reqLoop:
	for time.Since(startTime) < maxHTTPRequestTimeout {
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
		if resp.StatusCode == http.StatusServiceUnavailable && !slices.Contains(opts.expectedHTTPCodes, http.StatusServiceUnavailable) {
			if err := discardRespBody(resp); err != nil {
				return nil, err
			}
			if tryNum > shortRetriesOn503Amount {
				time.Sleep(longRetryOn503Delay)
			} else {
				time.Sleep(shortRetryOn503Delay)
			}
			logger.Verbose("503. retrying...")
			tryNum++
			continue
		}
		break
	}
	isCodeExpected := slices.Contains(opts.expectedHTTPCodes, resp.StatusCode)
	if isCodeExpected && opts.discardResp {
		err := discardRespBody(resp)
		return nil, err
	}
	httpResponse := &HTTPResponse{
		HTTPResp:             resp,
		expectedSysErrorCode: opts.expectedSysErrorCode,
		expectedHTTPCodes:    opts.expectedHTTPCodes,
	}
	if isCodeExpected && opts.responseHandler != nil {
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
	if resp.StatusCode != http.StatusOK && len(opts.expectedErrorContains) > 0 {
		sysError := map[string]interface{}{}
		if err := json.Unmarshal([]byte(respBody), &sysError); err != nil {
			return nil, err
		}
		actualError := sysError["sys.Error"].(map[string]interface{})["Message"].(string)
		if !containsAllMessages(opts.expectedErrorContains, actualError) {
			return nil, fmt.Errorf(`actual error message "%s" does not contain the expected messages %v`, actualError, opts.expectedErrorContains)
		}
	}
	return httpResponse, statusErr
}

func (c *implIHTTPClient) CloseIdleConnections() {
	c.client.CloseIdleConnections()
}

func containsAllMessages(strs []string, toFind string) bool {
	for _, str := range strs {
		if !strings.Contains(toFind, str) {
			return false
		}
	}
	return true
}

func (resp *HTTPResponse) ExpectedSysErrorCode() int {
	return resp.expectedSysErrorCode
}

func (resp *HTTPResponse) ExpectedHTTPCodes() []int {
	return resp.expectedHTTPCodes
}

func (resp *HTTPResponse) Println() {
	log.Println(resp.Body)
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
	return len(resp.Sections) == 0
}

func (fe FuncError) Error() string {
	if len(fe.ExpectedHTTPCodes) == 1 && fe.ExpectedHTTPCodes[0] == http.StatusOK {
		return fmt.Sprintf("status %d: %s", fe.HTTPStatus, fe.Message)
	}
	return fmt.Sprintf("status %d, expected %v: %s", fe.HTTPStatus, fe.ExpectedHTTPCodes, fe.Message)
}

func (fe FuncError) Unwrap() error {
	return fe.SysError
}

type implIHTTPClient struct {
	client *http.Client
}

func NewIHTTPClient() (client IHTTPClient, clenup func()) {
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
	client = &implIHTTPClient{client: &http.Client{Transport: tr}}
	return client, client.CloseIdleConnections
}
