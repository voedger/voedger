/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package blobprocessor

import (
	"context"
	"io"
	"net/http"
	"net/url"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iprocbus"
	"github.com/voedger/voedger/pkg/istructs"
)

type IRequestHandler interface {
	// false -> service unavailable
	Handle(appQName appdef.AppQName, wsid istructs.WSID, header http.Header, requestCtx context.Context,
		urlQueryValues url.Values, okResponseIniter func(headersKeyValue ...string) io.Writer, reader io.ReadCloser, errorResponder ErrorResponder) bool
}

// implemented in e.g. router
type ErrorResponder func(ststusCode int, args ...interface{})

type implIRequestHandler struct {
	procbus      iprocbus.IProcBus
	chanGroupIdx BLOBServiceChannelGroupIdx
}

func (r *implIRequestHandler) Handle(appQName appdef.AppQName, wsid istructs.WSID, header http.Header, requestCtx context.Context,
	urlQueryValues url.Values, okResponseIniter func(headersKeyValue ...string) io.Writer, reader io.ReadCloser, errorResponder ErrorResponder) bool {
	msg := &implIBLOBMessage{
		appQName:         appQName,
		wsid:             wsid,
		header:           header,
		requestCtx:       requestCtx,
		urlQueryValues:   urlQueryValues,
		okResponseIniter: okResponseIniter,
		reader:           reader,
		errorResponder:   errorResponder,
		done:             make(chan interface{}),
	}
	if success := r.procbus.Submit(uint(r.chanGroupIdx), 0, msg); !success {
		return false
	}
	<-msg.done
	return true
}

func NewIRequestHandler(procbus iprocbus.IProcBus, chanGroupIdx BLOBServiceChannelGroupIdx) IRequestHandler {
	return &implIRequestHandler{
		procbus:      procbus,
		chanGroupIdx: chanGroupIdx,
	}
}
