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
	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/iblobstorage"
	"github.com/voedger/voedger/pkg/iprocbus"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/pipeline"
)

type blobWorkpiece struct {
	pipeline.IWorkpiece
	blobMessage      iBLOBMessage_Base
	blobMessageWrite IBLOBMessage_Write
	blobMessageRead  IBLOBMessage_Read
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
	reader           io.Reader
	registerFuncName string
	resultErr        error
}

type implIBLOBMessage_base struct {
	appQName         appdef.AppQName
	wsid             istructs.WSID
	header           http.Header
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

func (m *implIBLOBMessage_base) AppQName() appdef.AppQName {
	return m.appQName
}

func (m *implIBLOBMessage_base) WSID() istructs.WSID {
	return m.wsid
}

func (m *implIBLOBMessage_base) Header() http.Header {
	return m.header
}

func (m *implIBLOBMessage_base) RequestCtx() context.Context {
	return m.requestCtx
}

func (m *implIBLOBMessage_Write) URLQueryValues() url.Values {
	return m.urlQueryValues
}

func (m *implIBLOBMessage_base) InitOKResponse(headersKeyValue ...string) io.Writer {
	return m.okResponseIniter(headersKeyValue...)
}

func (m *implIBLOBMessage_Write) Reader() io.ReadCloser {
	return m.reader
}

func (m *implIBLOBMessage_base) ReplyError(statusCode int, args ...any) {
	m.errorResponder(statusCode, args)
}

func (m *implIBLOBMessage_Read) ExistingBLOBIDOrSUUID() string {
	return m.existingBLOBIDOrSUUID
}

func (m *implIBLOBMessage_base) Release() {
	close(m.done)
}

func (m *implIBLOBMessage_base) RequestSender() bus.IRequestSender {
	return m.requestSender
}

type WLimiterFactory func() iblobstorage.WLimiterType

type iBLOBMessage_Base interface {
	pipeline.IWorkpiece
	AppQName() appdef.AppQName
	WSID() istructs.WSID
	Header() http.Header
	InitOKResponse(headersKeyValue ...string) io.Writer
	RequestCtx() context.Context
	ReplyError(statusCode int, args ...any)
	RequestSender() bus.IRequestSender
}

type IBLOBMessage_Read interface {
	iBLOBMessage_Base
	ExistingBLOBIDOrSUUID() string
}

type IBLOBMessage_Write interface {
	iBLOBMessage_Base
	Reader() io.ReadCloser
	URLQueryValues() url.Values
}

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
	b.blobMessage.Release()
}
