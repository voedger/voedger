/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package httpu

import (
	"fmt"
	"io"
	"net/http"
	"slices"
	"time"
)

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
func WithAuthorizeBy(token string) ReqOptFunc {
	return func(opts IReqOpts) {
		opts.httpOpts().headers[Authorization] = BearerPrefix + token
	}
}

func WithRespectRetryAfter() RetryOnStatusOpt {
	return func(policy *retryOnStatus) {
		policy.respectRetryAfter = true
	}
}

// WithRetryOnStatus returns a RetryPolicyOpt that can be used with WithRetryPolicy,
// or a ReqOptFunc that can be used directly with request options.
// default max 1 minute
func WithRetryOnStatus(statusCode int, retryOpts ...RetryOnStatusOpt) RetryPolicyOpt {
	return func(opts IReqOpts) {
		policy := retryOnStatus{
			statusCode:       statusCode,
			maxRetryDuration: defaultMaxRetryDuration,
		}
		for _, opt := range retryOpts {
			opt(&policy)
		}
		opts.httpOpts().retryOnStatus = append(opts.httpOpts().retryOnStatus, policy)
	}
}

func WithMaxRetryDuration(statusCode int, maxRetryDuration time.Duration) ReqOptFunc {
	return func(opts IReqOpts) {
		for i, policy := range opts.httpOpts().retryOnStatus {
			if policy.statusCode == statusCode {
				opts.httpOpts().retryOnStatus[i].maxRetryDuration = maxRetryDuration
				return
			}
		}
		panic(fmt.Sprintf("max retry duration for status code %d is specified but retry policy is not", statusCode))
	}
}

func WithDefaultAuthorize(principalToken string) ReqOptFunc {
	return func(opts IReqOpts) {
		if _, ok := opts.httpOpts().headers[Authorization]; !ok {
			opts.httpOpts().headers[Authorization] = BearerPrefix + principalToken
		}
	}
}

func WithURLPath(urlPath string) ReqOptFunc {
	return func(opts IReqOpts) {
		opts.httpOpts().urlPath = urlPath
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

func WithRetryOnError(matcher func(err error) (retry bool)) RetryPolicyOpt {
	return func(opts IReqOpts) {
		opts.httpOpts().retryOnErr = append(opts.httpOpts().retryOnErr, matcher)
	}
}

// WithRetryPolicy replaces the default retry policy with a custom one.
// It accepts only RetryPolicyOpt functions (e.g., WithRetryOnStatus, WithSkipRetryOnStatus, WithMaxRetryDurationOnStatus).
// To disable all retries, use WithNoRetryPolicy() instead.
func WithRetryPolicy(policyOpts ...RetryPolicyOpt) ReqOptFunc {
	return func(opts IReqOpts) {
		o := opts.httpOpts()
		o.retryOnStatus = nil
		for _, policy := range policyOpts {
			policy(opts)
		}
	}
}

// WithNoRetryPolicy disables all retry policies (both default and custom).
// This is useful when you want to make requests without any automatic retries.
func WithNoRetryPolicy() ReqOptFunc {
	return WithRetryPolicy()
}

func WithDefaultRetryPolicy() ReqOptFunc {
	return WithRetryPolicy(DefaultRetryPolicyOpts...)
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

func WithCustomOpts(key any, value any) ReqOptFunc {
	return func(opts IReqOpts) {
		opts.httpOpts().customOpts[key] = value
	}
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

func (opts *reqOpts) CustomOpts(key any) (customOpts any) {
	return opts.customOpts[key]
}
