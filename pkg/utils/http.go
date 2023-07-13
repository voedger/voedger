/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package coreutils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	ibus "github.com/untillpro/airs-ibus"
	"github.com/untillpro/goutils/logger"
	"golang.org/x/exp/slices"

	"github.com/voedger/voedger/pkg/istructs"
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

func ReplyErrf(bus ibus.IBus, sender interface{}, status int, args ...interface{}) {
	ReplyErrDef(bus, sender, NewHTTPErrorf(status, args...), http.StatusInternalServerError)
}

func ReplyErrDef(bus ibus.IBus, sender interface{}, err error, defaultStatusCode int) {
	res := WrapSysError(err, defaultStatusCode).(SysError)
	ReplyJSON(bus, sender, res.HTTPStatus, res.ToJSON())
}

func ReplyErr(bus ibus.IBus, sender interface{}, err error) {
	ReplyErrDef(bus, sender, err, http.StatusInternalServerError)
}

func ReplyJSON(bus ibus.IBus, sender interface{}, httpCode int, body string) {
	bus.SendResponse(sender, ibus.Response{
		ContentType: ApplicationJSON,
		StatusCode:  httpCode,
		Data:        []byte(body),
	})
}

func ReplyBadRequest(bus ibus.IBus, sender interface{}, message string) {
	ReplyErrf(bus, sender, http.StatusBadRequest, message)
}

func replyAccessDenied(bus ibus.IBus, sender interface{}, code int, message string) {
	msg := "access denied"
	if len(message) > 0 {
		msg += ": " + message
	}
	ReplyErrf(bus, sender, code, msg)
}

func ReplyAccessDeniedUnauthorized(bus ibus.IBus, sender interface{}, message string) {
	replyAccessDenied(bus, sender, http.StatusUnauthorized, message)
}

func ReplyAccessDeniedForbidden(bus ibus.IBus, sender interface{}, message string) {
	replyAccessDenied(bus, sender, http.StatusForbidden, message)
}

func ReplyUnauthorized(bus ibus.IBus, sender interface{}, message string) {
	ReplyErrf(bus, sender, http.StatusUnauthorized, message)
}

func ReplyInternalServerError(bus ibus.IBus, sender interface{}, message string, err error) {
	ReplyErrf(bus, sender, http.StatusInternalServerError, message, ": ", err)
}

// WithResponseHandler, WithLongPolling and WithDiscardResponse are mutual exclusive
func WithResponseHandler(responseHandler func(httpResp *http.Response)) ReqOptFunc {
	return func(ro *reqOpts) {
		ro.responseHandler = responseHandler
	}
}

// WithLongPolling, WithResponseHandler and WithDiscardResponse are mutual exclusive
func WithLongPolling() ReqOptFunc {
	return func(ro *reqOpts) {
		ro.responseHandler = func(resp *http.Response) {
			if !slices.Contains(ro.expectedHTTPCodes, resp.StatusCode) {
				body, err := readBody(resp)
				if err != nil {
					panic("failed to read response body in custom response handler: " + err.Error())
				}
				panic(fmt.Sprintf("actual status code %d, expected %v. Body: %s", resp.StatusCode, ro.expectedHTTPCodes, body))
			}
		}
	}
}

// WithDiscardResponse, WithResponseHandler and WithLongPolling are mutual exclusive
// causes *HTTPResponse to be nil
func WithDiscardResponse() ReqOptFunc {
	return func(opts *reqOpts) {
		opts.discardResp = true
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

func WithExpectedCode(expectedHTTPCode int) ReqOptFunc {
	return func(po *reqOpts) {
		po.expectedHTTPCodes = append(po.expectedHTTPCodes, expectedHTTPCode)
	}
}

// has priority over WithAuthorizeByIfNot
func WithAuthorizeBy(principalToken string) ReqOptFunc {
	return func(po *reqOpts) {
		po.headers[Authorization] = BearerPrefix + principalToken
	}
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

func ExpectSysError500() ReqOptFunc {
	return func(opts *reqOpts) {
		opts.expectedSysErrorCode = http.StatusInternalServerError
	}
}

type reqOpts struct {
	method            string
	headers           map[string]string
	cookies           map[string]string
	expectedHTTPCodes []int

	// used if no errors and an expected status code is received
	responseHandler func(httpResp *http.Response)

	timeoutMs            int64
	relativeURL          string
	discardResp          bool
	expectedSysErrorCode int
}

func req(url string, body string, client *http.Client, opts *reqOpts) (*http.Response, error) {
	req, err := http.NewRequest(opts.method, url, bytes.NewReader([]byte(body)))
	if err != nil {
		return nil, fmt.Errorf("NewRequest() failed: %w", err)
	}
	req.Close = true
	for k, v := range opts.headers {
		req.Header.Add(k, v)
	}
	for k, v := range opts.cookies {
		req.AddCookie(&http.Cookie{
			Name:  k,
			Value: v,
		})
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request do() failed: %w", err)
	}
	return resp, nil
}

// wrapped ErrUnexpectedStatusCode is returned -> *HTTPResponse contains a valid response body
// otherwise if err != nil (e.g. socket error)-> *HTTPResponse is nil
func FederationPOST(federationUrl *url.URL, relativeURL string, body string, optFuncs ...ReqOptFunc) (*HTTPResponse, error) {
	optFuncs = append(optFuncs, WithMethod(http.MethodPost))
	return FederationReq(federationUrl, relativeURL, body, optFuncs...)
}

func FederationReq(federationUrl *url.URL, relativeURL string, body string, optFuncs ...ReqOptFunc) (*HTTPResponse, error) {
	url := federationUrl.String() + "/" + relativeURL
	return Req(url, body, optFuncs...)
}

// status code expected -> DiscardBody, ResponseHandler are used
// status code is unexpected -> DiscardBody, ResponseHandler are ignored, body is read out, wrapped ErrUnexpectedStatusCode is returned
func Req(urlStr string, body string, optFuncs ...ReqOptFunc) (*HTTPResponse, error) {
	opts := &reqOpts{
		headers:   map[string]string{},
		cookies:   map[string]string{},
		timeoutMs: math.MaxInt,
		method:    http.MethodGet,
	}
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
	client := &http.Client{}
	var resp *http.Response
	var err error
	deadline := time.UnixMilli(opts.timeoutMs)
	tryNum := 0
	for time.Now().Before(deadline) {
		resp, err = req(urlStr, body, client, opts)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode == http.StatusServiceUnavailable && !slices.Contains(opts.expectedHTTPCodes, http.StatusServiceUnavailable) {
			if err := discardRespBody(resp); err != nil {
				return nil, err
			}
			if tryNum > shortRetriesAmount {
				time.Sleep(longRetryDelay)
			} else {
				time.Sleep(shortRetryDelay)
			}
			logger.Verbose("503. retrying...")
			tryNum++
			continue
		}
		break
	}
	httpResponse := &HTTPResponse{
		HTTPResp:             resp,
		expectedSysErrorCode: opts.expectedSysErrorCode,
		expectedHTTPCodes:    opts.expectedHTTPCodes,
	}
	isCodeExpected := slices.Contains(opts.expectedHTTPCodes, resp.StatusCode)
	if isCodeExpected {
		if opts.responseHandler != nil {
			opts.responseHandler(resp)
			return httpResponse, nil
		}
		if opts.discardResp {
			if err := discardRespBody(resp); err != nil {
				return nil, err
			}
			return nil, nil
		}
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

func FederationFunc(federationUrl *url.URL, relativeURL string, body string, optFuncs ...ReqOptFunc) (*FuncResponse, error) {
	httpResp, err := FederationPOST(federationUrl, relativeURL, body, optFuncs...)
	isUnexpectedCode := errors.Is(err, ErrUnexpectedStatusCode)
	if err != nil && !isUnexpectedCode {
		return nil, err
	}
	if httpResp == nil {
		return nil, nil
	}
	if isUnexpectedCode {
		m := map[string]interface{}{}
		if err = json.Unmarshal([]byte(httpResp.Body), &m); err != nil {
			return nil, err
		}
		if httpResp.HTTPResp.StatusCode == http.StatusOK {
			return nil, FuncError{
				SysError: SysError{
					HTTPStatus: http.StatusOK,
				},
				ExpectedHTTPCodes: httpResp.expectedHTTPCodes,
			}
		}
		sysErrorMap := m["sys.Error"].(map[string]interface{})
		return nil, FuncError{
			SysError: SysError{
				HTTPStatus: int(sysErrorMap["HTTPStatus"].(float64)),
				Message:    sysErrorMap["Message"].(string),
			},
			ExpectedHTTPCodes: httpResp.expectedHTTPCodes,
		}
	}
	res := &FuncResponse{
		HTTPResponse: httpResp,
		NewIDs:       map[string]int64{},
		CmdResult:    map[string]interface{}{},
	}
	if len(httpResp.Body) == 0 {
		return res, nil
	}
	if err = json.Unmarshal([]byte(httpResp.Body), &res); err != nil {
		return nil, err
	}
	if res.SysError.HTTPStatus > 0 && res.expectedSysErrorCode > 0 && res.expectedSysErrorCode != res.SysError.HTTPStatus {
		return nil, fmt.Errorf("sys.Error actual status %d, expected %v: %s", res.SysError.HTTPStatus, res.expectedSysErrorCode, res.SysError.Message)
	}
	return res, nil
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
	resp.Body.Close()
	return string(respBody), err
}

func discardRespBody(resp *http.Response) error {
	_, err := io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if err != nil {
		return fmt.Errorf("failed to discard response body: %w", err)
	}
	return nil
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
func (resp *FuncResponse) NewID() int64 {
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

type implIFederation struct {
	federationURL func() *url.URL
}

func (f *implIFederation) POST(appQName istructs.AppQName, wsid istructs.WSID, fn string, body string, opts ...ReqOptFunc) (*HTTPResponse, error) {
	return FederationPOST(f.federationURL(), fmt.Sprintf(`api/%s/%d/%s`, appQName, wsid, fn), body, opts...)
}

func (f *implIFederation) URL() *url.URL {
	return f.federationURL()
}

func NewIFederation(federationURL func() *url.URL) IFederation {
	return &implIFederation{federationURL: federationURL}
}
