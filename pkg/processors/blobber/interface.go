/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package blobprocessor

import (
	"context"
	"io"
	"net/url"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/iblobstorage"
	"github.com/voedger/voedger/pkg/istructs"
)

type IRequestHandler interface {
	// false -> service unavailable
	HandleRead(appQName appdef.AppQName, wsid istructs.WSID, header map[string]string, requestCtx context.Context,
		okResponseIniter func(headersKeyValue ...string) io.Writer,
		errorResponder ErrorResponder, existingBLOBIDOrSUUID string, requestSender bus.IRequestSender) bool
	HandleWrite(appQName appdef.AppQName, wsid istructs.WSID, header map[string]string, requestCtx context.Context,
		urlQueryValues url.Values, okResponseIniter func(headersKeyValue ...string) io.Writer, reader io.ReadCloser,
		errorResponder ErrorResponder, requestSender bus.IRequestSender) bool
	HandleWrite_V2(appQName appdef.AppQName, wsid istructs.WSID, header map[string]string, requestCtx context.Context,
		okResponseIniter func(headersKeyValue ...string) io.Writer, reader io.ReadCloser,
		errorResponder ErrorResponder, requestSender bus.IRequestSender, ownerRecord appdef.QName, ownerRecordField string) bool
	HandleWriteTemp_V2(appQName appdef.AppQName, wsid istructs.WSID, header map[string]string, requestCtx context.Context,
		okResponseIniter func(headersKeyValue ...string) io.Writer, reader io.ReadCloser,
		errorResponder ErrorResponder, requestSender bus.IRequestSender) bool
	HandleRead_V2(appQName appdef.AppQName, wsid istructs.WSID, header map[string]string, requestCtx context.Context,
		okResponseIniter func(headersKeyValue ...string) io.Writer,
		errorResponder ErrorResponder, ownerRecord appdef.QName, ownerRecordField string, ownerID istructs.RecordID,
		requestSender bus.IRequestSender) bool
	HandleReadTemp_V2(appQName appdef.AppQName, wsid istructs.WSID, header map[string]string, requestCtx context.Context,
		okResponseIniter func(headersKeyValue ...string) io.Writer,
		errorResponder ErrorResponder, requestSender bus.IRequestSender, suuid iblobstorage.SUUID) bool
}

// implemented in e.g. router package
type ErrorResponder func(ststusCode int, args ...interface{})
