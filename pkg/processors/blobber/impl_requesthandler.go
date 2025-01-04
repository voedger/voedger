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
	HandleRead(appQName appdef.AppQName, wsid istructs.WSID, header http.Header, requestCtx context.Context,
		okResponseIniter func(headersKeyValue ...string) io.Writer,
		errorResponder ErrorResponder, existingBLOBIDOrSUUID string) bool
	HandleWrite(appQName appdef.AppQName, wsid istructs.WSID, header http.Header, requestCtx context.Context,
		urlQueryValues url.Values, okResponseIniter func(headersKeyValue ...string) io.Writer, reader io.ReadCloser,
		errorResponder ErrorResponder) bool
}

// implemented in e.g. router package
type ErrorResponder func(ststusCode int, args ...interface{})

type implIRequestHandler struct {
	procbus      iprocbus.IProcBus
	chanGroupIdx BLOBServiceChannelGroupIdx
}

func (r *implIRequestHandler) HandleRead(appQName appdef.AppQName, wsid istructs.WSID, header http.Header, requestCtx context.Context,
	okResponseIniter func(headersKeyValue ...string) io.Writer,
	errorResponder ErrorResponder, existingBLOBIDOrSUUID string) bool {
	doneCh := make(chan interface{})
	return r.handle(&implIBLOBMessage_Read{
		implIBLOBMessage_base: implIBLOBMessage_base{
			appQName:         appQName,
			wsid:             wsid,
			header:           header,
			requestCtx:       requestCtx,
			okResponseIniter: okResponseIniter,
			errorResponder:   errorResponder,
			done:             doneCh,
		},
		existingBLOBIDOrSUUID: existingBLOBIDOrSUUID,
	}, doneCh)
}

func (r *implIRequestHandler) HandleWrite(appQName appdef.AppQName, wsid istructs.WSID, header http.Header, requestCtx context.Context,
	urlQueryValues url.Values, okResponseIniter func(headersKeyValue ...string) io.Writer, reader io.ReadCloser,
	errorResponder ErrorResponder) bool {
	doneCh := make(chan interface{})
	return r.handle(&implIBLOBMessage_Write{
		implIBLOBMessage_base: implIBLOBMessage_base{
			appQName:         appQName,
			wsid:             wsid,
			header:           header,
			requestCtx:       requestCtx,
			okResponseIniter: okResponseIniter,
			errorResponder:   errorResponder,
			done:             doneCh,
		},
		urlQueryValues: urlQueryValues,
		reader:         reader,
	}, doneCh)
}

func (r *implIRequestHandler) handle(msg any, doneCh <-chan interface{}) bool {
	if success := r.procbus.Submit(uint(r.chanGroupIdx), 0, msg); !success {
		return false
	}
	<-doneCh
	return true
}
