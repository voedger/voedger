/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
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
	"github.com/voedger/voedger/pkg/iprocbus"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/pipeline"
)

type blobWorkpiece struct {
	pipeline.IWorkpiece
	blobMessage      any
	blobMessageWrite *implIBLOBMessage_Write
	blobMessageRead  *implIBLOBMessage_Read
	duration         iblobstorage.DurationType
	nameQuery        []string
	mimeTypeQuery    []string
	ttl              string
	descr            iblobstorage.DescrType
	mediaType        string
	boundary         string
	contentType      string
	newBLOBID        istructs.RecordID
	newSUUID         iblobstorage.SUUID
	blobState        iblobstorage.BLOBState
	blobKey          iblobstorage.IBLOBKey
	writer           io.Writer
	registerFuncName string
	resultErr        error
}

type implIBLOBMessage_base struct {
	appQName         appdef.AppQName
	wsid             istructs.WSID
	header           map[string]string
	requestCtx       context.Context
	okResponseIniter func(headersKeyValue ...string) io.Writer
	errorResponder   ErrorResponder
	done             chan interface{}
	requestSender    bus.IRequestSender
}

type implIBLOBMessage_Read struct {
	implIBLOBMessage_base
	existingBLOBIDOrSUUID string
}

type implIBLOBMessage_Write struct {
	implIBLOBMessage_base
	reader         io.ReadCloser
	urlQueryValues url.Values
}

type WLimiterFactory func() iblobstorage.WLimiterType

type BLOBServiceChannel iprocbus.ServiceChannel

type BLOBServiceChannelGroupIdx uint

type badRequestWrapper struct {
	pipeline.NOOP
}

type sendWriteResult struct {
	pipeline.NOOP
}

type catchReadError struct {
	pipeline.NOOP
}

type blobOpSwitch struct {
}

func (b *blobWorkpiece) Release() {
	b.blobMessage.(pipeline.IWorkpiece).Release()
}

type implIRequestHandler struct {
	procbus      iprocbus.IProcBus
	chanGroupIdx BLOBServiceChannelGroupIdx
}

func (m *implIBLOBMessage_base) Release() {
	close(m.done)
}
