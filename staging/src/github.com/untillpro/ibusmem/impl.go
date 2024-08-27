/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package ibusmem

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/voedger/voedger/pkg/goutils/logger"

	ibus "github.com/voedger/voedger/staging/src/github.com/untillpro/airs-ibus"
)

// if ctx.Done() and SendParallelResponse simultaneously then return sections channel + err = ctx.Err()
// nobody reads the sections channel in this case (according to the IBus contract) so `case ctx.Done()` in trySendSection() will fire for sure
func (b *bus) SendRequest2(clientCtx context.Context, request ibus.Request, timeout time.Duration) (res ibus.Response, sections <-chan ibus.ISection, secError *error, err error) {
	wg := sync.WaitGroup{}
	handlerPanic := make(chan interface{}, 1)
	s := &channelSender{
		c:         make(chan interface{}, 1),
		timeout:   timeout,
		clientCtx: clientCtx,
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case result := <-s.c:
			switch result := result.(type) {
			case ibus.Response:
				res = result
			case *resultSenderClosable:
				rsender := result
				sections = rsender.sections
				secError = rsender.err
			}
			err = clientCtx.Err() // to make ctx.Done() take priority
		case <-clientCtx.Done():
			if err = checkPanic(handlerPanic); err == nil {
				err = clientCtx.Err()
			}
		case <-b.timerResponse(timeout):
			if err = checkPanic(handlerPanic); err == nil {
				err = ibus.ErrBusTimeoutExpired
			}
		case rIntf := <-handlerPanic:
			err = handlePanic(rIntf)
		}
	}()
	sender := NewISender(b, s)
	func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("handler panic:", fmt.Sprint(r), "\n", string(debug.Stack()))
				// will process panic in the goroutine instead of update err here to avoid data race
				// https://dev.untill.com/projects/#!607751
				handlerPanic <- r
			}
		}()
		b.requestHandler(clientCtx, sender, request)
	}()
	wg.Wait()
	return res, sections, secError, err
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

func checkPanic(ch <-chan interface{}) error {
	select {
	case r := <-ch:
		return handlePanic(r)
	default:
		return nil
	}
}

func (b *bus) SendResponse(sender interface{}, response ibus.Response) {
	s := sender.(*channelSender)
	s.send(response)
}

func (b *bus) SendParallelResponse2(sender interface{}) (rsender ibus.IResultSenderClosable) {
	s := sender.(*channelSender)
	var err error
	rsender = &resultSenderClosable{
		sections:     make(chan ibus.ISection),
		err:          &err,
		timeout:      s.timeout,
		clientCtx:    s.clientCtx,
		timerSection: b.timerSection,
		timerElement: b.timerElement,
	}
	s.send(rsender)
	return rsender
}

func (s *channelSender) send(value interface{}) {
	s.c <- value
	close(s.c)
}

func (s *resultSenderClosable) StartArraySection(sectionType string, path []string) {
	s.currentSection = arraySection{
		sectionType: sectionType,
		path:        path,
		elems:       s.updateElemsChannel(),
	}
}

func (s *resultSenderClosable) StartMapSection(sectionType string, path []string) {
	s.currentSection = mapSection{
		sectionType: sectionType,
		path:        path,
		elems:       s.updateElemsChannel(),
	}
}

func (s *resultSenderClosable) ObjectSection(sectionType string, path []string, element interface{}) (err error) {
	s.currentSection = &objectSection{
		sectionType: sectionType,
		path:        path,
		elements:    s.updateElemsChannel(),
	}
	err = s.SendElement("", element)
	s.elements = nil
	return
}

func (s *resultSenderClosable) SendElement(name string, el interface{}) (err error) {
	if el == nil {
		return nil
	}
	if s.elements == nil {
		panic("section is not started")
	}
	bb, ok := el.([]byte)
	if !ok {
		if bb, err = json.Marshal(el); err != nil {
			return
		}
	}
	if err = s.tryToSendSection(); err != nil {
		return
	}
	element := element{
		name:  name,
		value: bb,
	}
	return s.tryToSendElement(element)
}

func (s *resultSenderClosable) Close(err error) {
	*s.err = err
	close(s.sections)
	if s.elements != nil {
		close(s.elements)
	}
}

func (s *resultSenderClosable) updateElemsChannel() chan element {
	if s.elements != nil {
		close(s.elements)
	}
	s.elements = make(chan element)
	return s.elements
}

func (s *resultSenderClosable) tryToSendSection() (err error) {
	if s.currentSection != nil {
		select {
		case s.sections <- s.currentSection:
			s.currentSection = nil
			return s.clientCtx.Err() // ctx.Done() has priority on simultaneous (s.ctx.Done() and s.sections<- success)
		case <-s.clientCtx.Done():
			return s.clientCtx.Err()
		case <-s.timerSection(s.timeout):
			return ibus.ErrNoConsumer
		}
	}
	return nil
}

func (s *resultSenderClosable) tryToSendElement(value element) (err error) {
	select {
	case s.elements <- value:
		return s.clientCtx.Err() // ctx.Done() has priority on simultaneous (s.ctx.Done() and s.elemets<- success)
	case <-s.clientCtx.Done():
		return s.clientCtx.Err()
	case <-s.timerElement(s.timeout):
		return ibus.ErrNoConsumer
	}
}

func (s arraySection) Type() string {
	return s.sectionType
}

func (s arraySection) Path() []string {
	return s.path
}

func (s arraySection) Next(ctx context.Context) (value []byte, ok bool) {
	select {
	case e, ok := <-s.elems:
		if ok && ctx.Err() == nil {
			return e.value, true
		}
	case <-ctx.Done():
	}
	return nil, false
}

func (s mapSection) Type() string {
	return s.sectionType
}

func (s mapSection) Path() []string {
	return s.path
}

func (s mapSection) Next(ctx context.Context) (name string, value []byte, ok bool) {
	select {
	case e, ok := <-s.elems:
		if ok && ctx.Err() == nil {
			return e.name, e.value, true
		}
	case <-ctx.Done():
	}
	return "", nil, false
}

func (s *objectSection) Type() string {
	return s.sectionType
}

func (s *objectSection) Path() []string {
	return s.path
}

func (s *objectSection) Value(ctx context.Context) []byte {
	if !s.elementReceived {
		select {
		case e, ok := <-s.elements:
			if ok && ctx.Err() == nil {
				s.elementReceived = true
				return e.value
			}
		case <-ctx.Done():
		}
	}
	return nil
}

func (s *implISender) SendResponse(resp ibus.Response) {
	s.bus.SendResponse(s.sender, resp)
}

func (s *implISender) SendParallelResponse() ibus.IResultSenderClosable {
	return s.bus.SendParallelResponse2(s.sender)
}
