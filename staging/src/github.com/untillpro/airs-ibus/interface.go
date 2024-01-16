/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package ibus

import (
	"context"
	"time"
)

// Deprecated: use SendRequest2
// SendRequest used by router and app, sends a message to a given queue
// If err is not nil res and chunks are nil
// If chunks is not nil chunks must be read to the end
// Non-nil chunksError when chunks are closed means an error in chunks
// `err` and `error` can be a wrapped ErrTimeoutExpired (Checked as errors.Is(err, ErrTimeoutExpired))
// If chunks is not nil then Response must be ignored
var SendRequest func(ctx context.Context,
	request *Request, timeout time.Duration) (res *Response, chunks <-chan []byte, chunksError *error, err error)

// SendRequest2 used by router and app, sends a message to a given queue
// err is not nil -> NATS-related error occurred before or during reading the first response packet. Sections and secError are nil, res must be ignored
// timeout means timeout during reading Response or ISection
// sections not nil ->
//   - res must be ignored
//   - sections must be read to the end
//   - non-nil secError when sections are closed means NATS-related error during reading sections or an error came with IResultSenderCloseable.Close()
//   - `ctx.Done()` -> implementation will close `sections`. Also `I*Section.Next()` will return false
//
// sections is nil -> res and err only should be used as the result. secError is nil
// `err` and `*secError` can be a wrapped ErrTimeoutExpired (Checked as errors.Is(err, ErrTimeoutExpired))
var SendRequest2 func(ctx context.Context,
	request Request, timeout time.Duration) (res Response, sections <-chan ISection, secError *error, err error)

// RequestHandler used by app
var RequestHandler func(ctx context.Context, sender interface{}, request Request)

// SendResponse used by app
var SendResponse func(ctx context.Context, sender interface{}, response Response)

// Deprecated: use SendParallelResponce2
// SendParallelResponse s.e.
// If chunks is not nil they must be readed by implementation to the end
// Chunks must be closed by sender
// response is valid when chunks finishes or nil
// chunkError is set by implementation when it could not send chunk
var SendParallelResponse func(ctx context.Context, sender interface{}, chunks <-chan []byte, chunkError *error)

// SendParallelResponse2 ???
// Result of Close
var SendParallelResponse2 func(ctx context.Context, sender interface{}) (rsender IResultSenderClosable)

// Deprecated: use MetricCntSerialRequest
// MetricSerialRequestCnt s.e.
var MetricSerialRequestCnt uint64

// Deprecated: use MetricDurSerialRequest
// MetricSerialRequestDurNs s.e.
var MetricSerialRequestDurNs uint64

// MetricCntSerialRequest  number of serial requests
var MetricCntSerialRequest func(ctx context.Context) uint64

// MetricDurSerialRequest duration of serial requests
var MetricDurSerialRequest func(ctx context.Context) uint64

/*
Provider must take RequestHandler as a parameter

SendRequest2 creates a temp struct which is passed to RequestHandler as a `sender`
RequestHandler calls either SendResponse or SendParallelResponse2 using sender
SendRequest2 should read answer using sender structure

Simple SendRequest2 implementation
  - Create channel
  - Run goroutine which calls RequestHandler
  - Read from channel
  - Detect answer type - SendResponse or SendParallelResponse2
  - Return either Response or channel
*/
type IBus interface {
	// SendRequest is called by sender side
	// err != nil -> nothing made, skip all other results
	// sections nil
	// - secError is nil
	// - res only should be used
	// sections not nil
	// - res should not be used
	// - sections chan should be read out
	// - *secError should not be touched until sections emptied
	// - *secError must be checked for error right after sections emptied
	// behaviour on ctx.Done:
	// - caller of SendRequest2() should not check ctx.Done() on sections read. Sections chan will be closed by bus on IResultSenderClosable.Close()
	// - successful section element send and ctx.Done() happened simulateously -> SendElement() should return ctx.Err()
	// neither SendResponse nor SendParallelResponse2 called during timeout -> err is ibus.ErrTimeoutExpired
	SendRequest2(ctx context.Context, request Request, timeout time.Duration) (res Response, sections <-chan ISection, secError *error, err error)

	// SendResponse is called by service side to respond with a signle response.
	// SendResponse is called again or SendParallelResponse2 is called after -> panic
	SendResponse(sender interface{}, response Response)

	// SendParallelResponse2 is called by service side to respond with a sections response
	// SendParallelResponse2 is called again or SendResponse is called after -> panic
	// IResultSenderClosable.Close() must be called at the end
	SendParallelResponse2(sender interface{}) (rsender IResultSenderClosable)
}

type ISender interface {
	SendResponse(resp Response)
	SendParallelResponse() IResultSenderClosable
}