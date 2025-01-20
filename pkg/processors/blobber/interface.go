/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package blobprocessor

import (
	"context"
	"io"
	"net/http"
	"net/url"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/istructs"
)

type IRequestHandler interface {
	// false -> service unavailable
	HandleRead(appQName appdef.AppQName, wsid istructs.WSID, header http.Header, requestCtx context.Context,
		okResponseIniter func(headersKeyValue ...string) io.Writer,
		errorResponder ErrorResponder, existingBLOBIDOrSUUID string, requestSender bus.IRequestSender) bool
	HandleWrite(appQName appdef.AppQName, wsid istructs.WSID, header http.Header, requestCtx context.Context,
		urlQueryValues url.Values, okResponseIniter func(headersKeyValue ...string) io.Writer, reader io.ReadCloser,
		errorResponder ErrorResponder, requestSender bus.IRequestSender) bool
}

// implemented in e.g. router package
type ErrorResponder func(ststusCode int, args ...interface{})
