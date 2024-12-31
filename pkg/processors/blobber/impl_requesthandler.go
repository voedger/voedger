/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package blobprocessor

import "github.com/voedger/voedger/pkg/iprocbus"

type IRequestHandler interface {
	// false -> service unavailable
	Handle(msg interface{}) bool
}

// implemented in e.g. router
type ErrorResponder func(ststusCode int, args ...interface{})

type implIRequestHandler struct {
	procbus      iprocbus.IProcBus
	chanGroupIdx BLOBServiceChannelGroupIdx
}

func (r *implIRequestHandler) Handle(msg interface{}) bool {
	if success := r.procbus.Submit(uint(r.chanGroupIdx), 0, msg); !success {
		return false
	}

	// тут подождать завершения
	return true
}

func NewIRequestHandler(procbus iprocbus.IProcBus, chanGroupIdx BLOBServiceChannelGroupIdx) IRequestHandler {
	return &implIRequestHandler{
		procbus:      procbus,
		chanGroupIdx: chanGroupIdx,
	}
}
