/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package httpu

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"slices"
	"time"

	"github.com/voedger/voedger/pkg/goutils/logger"
	retrier "github.com/voedger/voedger/pkg/goutils/retry"
)

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

func (c *implIHTTPClient) req(ctx context.Context, urlStr string, body string, optFuncs ...ReqOptFunc) (*HTTPResponse, error) {
	opts := &reqOpts{
		headers: map[string]string{},
		cookies: map[string]string{},
		validators: []func(IReqOpts) (panicMessage string){
			optsValidator_responseHandling,
		},
		customOpts: map[any]any{},
	}
	for _, defaultOptFunc := range c.defaultOpts {
		defaultOptFunc(opts)
	}
	for _, optFunc := range optFuncs {
		optFunc(opts)
	}
	for _, optFunc := range opts.appendedOpts {
		optFunc(opts)
	}
	if len(opts.method) == 0 {
		opts.method = http.MethodGet
	}

	if len(opts.expectedHTTPCodes) == 0 {
		opts.expectedHTTPCodes = append(opts.expectedHTTPCodes, http.StatusOK, http.StatusCreated)
	}
	if len(opts.urlPath) > 0 {
		netURL, err := url.Parse(urlStr)
		if err != nil {
			return nil, err
		}
		netURL.Path = opts.urlPath
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
		for _, matcher := range opts.retryOnErr {
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

		for _, retryPolicy := range opts.retryOnStatus {
			if resp.StatusCode != retryPolicy.statusCode {
				continue
			}
			if retryPolicy.maxRetryDuration > 0 && time.Since(startTime) > retryPolicy.maxRetryDuration {
				break
			}
			if err := discardRespBody(resp); err != nil {
				return nil, err
			}
			if retryPolicy.respectRetryAfter {
				retryAfterDuration := parseRetryAfterHeader(resp)
				if retryAfterDuration > 0 {
					logger.Verbose("%d. retrying after %v...", resp.StatusCode, retryAfterDuration)
					// Sleep for the custom delay, respecting context cancellation
					select {
					case <-time.After(retryAfterDuration):
					case <-ctx.Done():
						return nil, ctx.Err()
					}
				}
			}
			logger.Verbose(resp.StatusCode, "retrying...")
			return nil, errRetry
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
		Opts:     opts,
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
