/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package bus

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"
	"sync"
	"time"

	"github.com/voedger/voedger/pkg/goutils/httpu"
	"github.com/voedger/voedger/pkg/goutils/logger"
)

func (rs *implIRequestSender) SendRequest(clientCtx context.Context, req Request) (responseCh <-chan any, responseMeta ResponseMeta, responseErr *error, err error) {
	respWriter := &implResponseWriter{
		ch:        make(chan any, 1), // buf size 1 to make single write on Respond()
		clientCtx: clientCtx,
		tm:        rs.tm,
		resultErr: new(error),
	}
	responder := &implIResponder{
		respWriter:     respWriter,
		responseMetaCh: make(chan ResponseMeta, 1),
	}
	handlerPanic := make(chan interface{})
	firstReponseReceived := make(chan struct{})
	startTime := time.Now()
	wg := sync.WaitGroup{}
	wg.Go(func() {
		defer wg.Done()
		warningTicker := time.NewTicker(noFirstResponseWarningInterval)
		defer warningTicker.Stop()
		for {
			select {
			case <-warningTicker.C:
				elapsed := time.Since(startTime)
				logger.Warning("no first response for", elapsed, "on", req.Resource)
			case <-firstReponseReceived:
				return
			}
		}
	})
	wg.Go(func() {
		defer wg.Done()
		select {
		case responseMeta = <-responder.responseMetaCh:
			err = clientCtx.Err()
		case <-clientCtx.Done():
			if err = checkHandlerPanic(handlerPanic); err == nil {
				err = clientCtx.Err()
			}
		case panicIntf := <-handlerPanic:
			err = handlePanic(panicIntf)
		}
		close(firstReponseReceived)
	})
	func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("handler panic:", fmt.Sprint(r), "\n", string(debug.Stack()))
				handlerPanic <- r
			}
		}()
		rs.requestHandler(clientCtx, req, responder)
	}()
	wg.Wait()
	return respWriter.ch, responseMeta, respWriter.resultErr, err
}

func checkHandlerPanic(ch <-chan interface{}) error {
	select {
	case r := <-ch:
		return handlePanic(r)
	default:
		return nil
	}
}

func handlePanic(r interface{}) error {
	switch rTyped := r.(type) {
	case string:
		return errors.New(rTyped)
	case error:
		return rTyped
	default:
		// notest
		return fmt.Errorf("%#v", r)
	}
}

func (r *implIResponder) StreamJSON(statusCode int) IResponseWriter {
	r.checkStarted()
	select {
	case r.responseMetaCh <- ResponseMeta{ContentType: httpu.ContentType_ApplicationJSON, StatusCode: statusCode}:
	default:
		// do nothing if no consumer already.
		// will get ErrNoConsumer on the next Write()
	}
	return r.respWriter
}

func (r *implIResponder) StreamEvents() IResponseWriter {
	r.checkStarted()
	responseMeta := ResponseMeta{
		ContentType: httpu.ContentType_TextEventStream,
		StatusCode:  http.StatusOK,
		mode:        RespondMode_StreamEvents,
	}
	select {
	case r.responseMetaCh <- responseMeta:
	default:
		// do nothing if no consumer already.
		// will get ErrNoConsumer on the next Write()
	}
	return r.respWriter
}

func (r *implIResponder) Respond(responseMeta ResponseMeta, obj any) error {
	r.checkStarted()
	if responseMeta.mode != 0 {
		panic("responseMeta.mode is set by someone else!")
	}
	responseMeta.mode = RespondMode_Single
	select {
	case r.responseMetaCh <- responseMeta:
		r.respWriter.ch <- obj // buf size 1
		close(r.respWriter.ch)
	case <-r.respWriter.clientCtx.Done():
		return r.respWriter.clientCtx.Err()
	}
	return nil
}

func (rs *implResponseWriter) Write(obj any) error {
	noConsumerTimerChan := rs.tm.NewTimerChan(noConsumerTimeout)
	select {
	case rs.ch <- obj:
	case <-rs.clientCtx.Done():
	case <-noConsumerTimerChan:
		return ErrNoConsumer
	}
	return rs.clientCtx.Err()
}

func (rs *implResponseWriter) Close(err error) {
	*rs.resultErr = err
	close(rs.ch)
}

func (r *implIResponder) checkStarted() {
	if r.started {
		panic("unable to start the response more than once")
	}
	r.started = true
}

func (r ResponseMeta) Mode() RespondMode {
	return r.mode
}
