/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Alisher Nurmanov
 */

package ihttpimpl

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/untillpro/goutils/logger"

	"github.com/voedger/voedger/pkg/ihttp"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

type Receiver func(processorsCtx context.Context, request interface{}, sectionsWriter SectionsWriterType) (response interface{}, status ihttp.Status, err error)

type Metrics struct {
	MaxNumOfConcurrentRequests int
	NumOfConcurrentRequests    int
}

type addressType struct {
	owner     string
	app       string
	partition int
	part      string
}

func (addr addressType) String() string {
	return fmt.Sprintf("addr: %v/%v/%v/%v", addr.owner, addr.app, addr.partition, addr.part)
}

type errSender struct {
	err    error
	status ihttp.Status
}

func (es *errSender) Send(ctx context.Context, msg interface{}, rectionHandler ihttp.SectionsHandlerType) (response interface{}, status ihttp.Status, err error) {
	return nil, es.status, es.err
}

type addressHandlerType struct {
	addr                addressType
	httpProc            *httpProcessor
	processorsCtx       context.Context
	processorsCtxCancel context.CancelFunc
	wg                  sync.WaitGroup
	requestChannel      requestChannelType
	numOfProcessors     int
}

func (ah *addressHandlerType) String() string {
	return fmt.Sprintf("%v, numOfProcessors: %v", ah.addr, ah.numOfProcessors)
}

func (ah *addressHandlerType) senderErr(request interface{}, e error) (response interface{}, status ihttp.Status, err error) {
	logger.Warning(e, ah.String(), ", request:", request)
	return NewResult(nil, e, ah.String(), "")
}

func (ah *addressHandlerType) Send(ctx context.Context, request interface{}, sectionsHandler ihttp.SectionsHandlerType) (response interface{}, status ihttp.Status, err error) {

	var requestContext *requestContextType

	select {
	case requestContext = <-ah.httpProc.requestContextsPool:
	default:
		logger.Warning(ErrBusUnavailable)
		return NewResult(nil, ErrBusUnavailable, "", "")
	}

	// cleanup requestContext
	{
		requestContext.refCount = 2
		defer func() {
			requestContext.release(ah.httpProc)
		}()

		requestContext.senderContext = ctx

		if !requestContext.errReached {
			logger.Warning("error was not reached", ah.String())
			requestContext.responseChannel = make(responseChannelType, ResponseChannelBufferSize)
		}

		requestContext.errReached = false

		requestContext.msg = request
	}

	select {
	case ah.requestChannel <- requestContext:
	default:
		requestContext.refCount = 1 // for correct defer
		return ah.senderErr(request, ErrServiceUnavailable)
	}

	for {
		coreutils.ResetTimer(requestContext.senderTimer, ah.httpProc.readWriteTimeout)
		select {
		case <-requestContext.senderTimer.C:
			if requestContext.responseChannelClosed {
				return ah.senderErr(request, ErrSlowClient)
			}
			return ah.senderErr(request, ErrReadTimeoutExpired)
		case response, ok := <-requestContext.responseChannel:
			if ctx.Err() != nil {
				return NewResult(nil, ErrClientClosedRequest, "", "")
			}
			if !ok {
				return ah.senderErr(request, ErrSlowClient)
			}
			err = response.err
			if err != nil {
				if err == errEOF {
					err = nil
				}
				requestContext.errReached = true
				return response.msg, response.status, err
			}
			sectionsHandler(response.msg)
		}
	}
}

type requestContextType struct {
	mu                    sync.Mutex
	refCount              int
	responseChannel       responseChannelType
	responseChannelClosed bool
	senderTimer           *time.Timer
	senderContext         context.Context
	msg                   interface{}
	errReached            bool
}

func (rc *requestContextType) release(httpProc *httpProcessor) {
	rc.mu.Lock()
	rc.refCount--
	if rc.refCount == 0 {
		httpProc.requestContextsPool <- rc
	}
	rc.mu.Unlock()
}

type responseType struct {
	msg    interface{}
	status ihttp.Status
	err    error
}

type responseChannelType chan responseType
type requestChannelType chan *requestContextType

type processor struct {
	ah             *addressHandlerType
	numOfResponses int
	requestContext *requestContextType
	processorTimer *time.Timer
}

func (p *processor) sendResponse(msg interface{}, status ihttp.Status, err error) bool {

	closeResponseChannel := func() bool {
		p.requestContext.responseChannelClosed = true
		close(p.requestContext.responseChannel)
		return false
	}

	if p.requestContext.responseChannelClosed {
		return false
	}
	if p.requestContext.senderContext.Err() != nil {
		logger.Info("sender context is closed")
		return closeResponseChannel()
	}

	response := responseType{msg: msg, status: status, err: err}

	if p.numOfResponses < ResponseChannelBufferSize {
		p.requestContext.responseChannel <- response
		p.numOfResponses += 1
		return true
	}

	coreutils.ResetTimer(p.processorTimer, p.ah.httpProc.readWriteTimeout)
	select {
	case <-p.ah.processorsCtx.Done():
		return closeResponseChannel()
	case <-p.processorTimer.C:
		logger.Warning(ErrSlowClient, p.ah.String())
		return closeResponseChannel()
	case p.requestContext.responseChannel <- response:
		return true
	}

}

func (p *processor) Write(section interface{}) bool {
	return p.sendResponse(section, ihttp.Status{}, nil)
}

func (p *processor) process(receiver Receiver, wg *sync.WaitGroup, ch requestChannelType) {

	defer wg.Done()
	for {
		select {
		case <-p.ah.processorsCtx.Done():
			return
		case p.requestContext = <-ch:
			p.numOfResponses = 0
			func() {
				defer p.requestContext.release(p.ah.httpProc)
				response, status, err := receiver(p.ah.processorsCtx, p.requestContext.msg, p)
				if err == nil {
					err = errEOF
				}
				p.sendResponse(response, status, err)
			}()

		}
	}
}

func NullHandler(responsePart interface{}) {}
