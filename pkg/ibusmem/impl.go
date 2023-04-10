/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package ibusmem

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/untillpro/goutils/logger"
	"github.com/untillpro/voedger/pkg/ibus"
	coreutils "github.com/untillpro/voedger/pkg/utils"
)

type bus struct {
	maxNumOfConcurrentRequests int
	mu                         sync.RWMutex
	readWriteTimeout           time.Duration
	addressHandlersMap         map[addressType]*addressHandlerType
	requestContextsPool        chan *requestContextType
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

func (rc *requestContextType) release(pbus *bus) {
	rc.mu.Lock()
	rc.refCount--
	if rc.refCount == 0 {
		pbus.requestContextsPool <- rc
	}
	rc.mu.Unlock()
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

type responseType struct {
	msg    interface{}
	status ibus.Status
	err    error
}

type requestChannelType chan *requestContextType
type responseChannelType chan responseType

type addressHandlerType struct {
	addr                addressType
	pbus                *bus
	processorsCtx       context.Context
	processorsCtxCancel context.CancelFunc
	wg                  sync.WaitGroup
	requestChannel      requestChannelType
	numOfProcessors     int
}

func (ah *addressHandlerType) String() string {
	return fmt.Sprintf("%v, numOfProcessors: %v", ah.addr, ah.numOfProcessors)
}

func (ah *addressHandlerType) senderErr(request interface{}, e error) (response interface{}, status ibus.Status, err error) {
	logger.Warning(e, ah.String(), ", request:", request)
	return ibus.NewResult(nil, e, ah.String(), "")
}

func (ah *addressHandlerType) Send(ctx context.Context, request interface{}, sectionsHandler ibus.SectionsHandlerType) (response interface{}, status ibus.Status, err error) {

	var requestContext *requestContextType

	select {
	case requestContext = <-ah.pbus.requestContextsPool:
	default:
		logger.Warning(ibus.ErrBusUnavailable)
		return ibus.NewResult(nil, ibus.ErrBusUnavailable, "", "")
	}

	// cleanup requestContext
	{
		requestContext.refCount = 2
		defer func() {
			requestContext.release(ah.pbus)
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
		return ah.senderErr(request, ibus.ErrServiceUnavailable)
	}

	for {
		coreutils.ResetTimer(requestContext.senderTimer, ah.pbus.readWriteTimeout)
		select {
		case <-requestContext.senderTimer.C:
			if requestContext.responseChannelClosed {
				return ah.senderErr(request, ibus.ErrSlowClient)
			}
			return ah.senderErr(request, ibus.ErrReadTimeoutExpired)
		case response, ok := <-requestContext.responseChannel:
			if ctx.Err() != nil {
				return ibus.NewResult(nil, ibus.ErrClientClosedRequest, "", "")
			}
			if !ok {
				return ah.senderErr(request, ibus.ErrSlowClient)
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

type processor struct {
	ah             *addressHandlerType
	numOfResponses int
	requestContext *requestContextType
	processorTimer *time.Timer
}

func (p *processor) sendResponse(msg interface{}, status ibus.Status, err error) bool {

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

	coreutils.ResetTimer(p.processorTimer, p.ah.pbus.readWriteTimeout)
	select {
	case <-p.ah.processorsCtx.Done():
		return closeResponseChannel()
	case <-p.processorTimer.C:
		logger.Warning(ibus.ErrSlowClient, p.ah.String())
		return closeResponseChannel()
	case p.requestContext.responseChannel <- response:
		return true
	}

}

func (p *processor) Write(section interface{}) bool {
	return p.sendResponse(section, ibus.Status{}, nil)
}

func (p *processor) process(receiver ibus.Receiver, wg *sync.WaitGroup, ch requestChannelType) {

	defer wg.Done()
	for {
		select {
		case <-p.ah.processorsCtx.Done():
			return
		case p.requestContext = <-ch:
			p.numOfResponses = 0
			func() {
				defer p.requestContext.release(p.ah.pbus)
				response, status, err := receiver(p.ah.processorsCtx, p.requestContext.msg, p)
				if err == nil {
					err = errEOF
				}
				p.sendResponse(response, status, err)
			}()

		}
	}
}

func (b *bus) RegisterReceiver(owner string, app string, partition int, part string, r ibus.Receiver, numOfProcessors, bufferSize int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	addr := addressType{owner, app, partition, part}
	if _, ok := b.addressHandlersMap[addr]; ok {
		panic(fmt.Sprintf("receivers already exists: %+v", addr))
	}

	ctx, cancel := context.WithCancel(context.Background())
	ah := addressHandlerType{
		addr:                addr,
		pbus:                b,
		processorsCtx:       ctx,
		processorsCtxCancel: cancel, wg: sync.WaitGroup{},
		requestChannel: make(requestChannelType, bufferSize), numOfProcessors: numOfProcessors,
	}

	for i := 0; i < numOfProcessors; i++ {
		ah.wg.Add(1)
		proc := processor{ah: &ah, processorTimer: time.NewTimer(b.readWriteTimeout)}
		go proc.process(r, &ah.wg, ah.requestChannel)
	}

	b.addressHandlersMap[addr] = &ah
	logger.Info("receiver registered:", &ah)

}

func (b *bus) UnregisterReceiver(owner string, app string, partition int, part string) (ok bool) {

	b.mu.RLock()
	addr := addressType{owner, app, partition, part}
	pReciever, ok := b.addressHandlersMap[addr]
	b.mu.RUnlock()

	if !ok {
		logger.Info("receiver not found:", fmt.Sprintf("%+v", addr))
		return false
	}

	pReciever.processorsCtxCancel()
	pReciever.wg.Wait()

	b.mu.Lock()
	delete(b.addressHandlersMap, addr)
	b.mu.Unlock()
	logger.Info("receiver unregistered:", pReciever)

	return true
}

func (b *bus) GetMetrics() (metrics ibus.Metrics) {
	metrics.MaxNumOfConcurrentRequests = b.maxNumOfConcurrentRequests
	metrics.NumOfConcurrentRequests = b.maxNumOfConcurrentRequests - len(b.requestContextsPool)
	return
}

type errSender struct {
	err    error
	status ibus.Status
}

func (es *errSender) Send(ctx context.Context, msg interface{}, rectionHandler ibus.SectionsHandlerType) (response interface{}, status ibus.Status, err error) {
	return nil, es.status, es.err
}

func (b *bus) QuerySender(owner string, app string, partition int, part string) (sender ibus.ISender, ok bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	addr := addressType{owner, app, partition, part}
	r, ok := b.addressHandlersMap[addr]
	if ok {
		return r, ok
	}

	es := errSender{}
	_, es.status, es.err = ibus.NewResult(nil, ibus.ErrReceiverNotFound, addr.String(), "")
	return &es, ok
}

func (b *bus) cleanup() {
	addrs := make([]addressType, 0, len(b.addressHandlersMap))
	for addr := range b.addressHandlersMap {
		addrs = append(addrs, addr)
	}
	if len(addrs) > 0 {
		logger.Warning("bus cleanup:", len(addrs), "address(es)")
	}
	for _, r := range addrs {
		b.UnregisterReceiver(r.owner, r.app, r.partition, r.part)
	}
	logger.Info("bus stopped")
}
