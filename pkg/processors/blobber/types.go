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
	"github.com/voedger/voedger/pkg/iblobstorage"
	"github.com/voedger/voedger/pkg/iprocbus"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/pipeline"
)

// type BLOBOperation int

// const (
// 	BLOBOperation_Null BLOBOperation = iota
// 	BLOBOperation_Read_Persistent
// 	BLOBOperation_Read_Temporary
// 	BLOBOperation_Write_Persistent_Single
// 	BLOBOperation_Write_Persistent_Multipart
// 	BLOBOperation_Write_Temporary_Single
// 	BLOBOperation_Write_Temporary_Multipart
// )

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

// func NewIBLOBMessage(appQName appdef.AppQName, wsid istructs.WSID, header http.Header, requestCtx context.Context,
// 	urlQueryValues url.Values, okResponseIniter func(headersKeyValue ...string) io.Writer, reader io.ReadCloser, errorResponder ErrorResponder) (msg IBLOBMessage, doneAwaiter func()) {
// 	msgImpl := &implIBLOBMessage{
// 		appQName:         appQName,
// 		wsid:             wsid,
// 		header:           header,
// 		requestCtx:       requestCtx,
// 		urlQueryValues:   urlQueryValues,
// 		okResponseIniter: okResponseIniter,
// 		reader:           reader,
// 		errorResponder:   errorResponder,
// 		done:             make(chan interface{}),
// 	}
// 	return msgImpl, func() {
// 	}
// }

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

type WLimiterFactory func() iblobstorage.WLimiterType

type iBLOBMessage_Base interface {
	pipeline.IWorkpiece
	AppQName() appdef.AppQName
	WSID() istructs.WSID
	Header() http.Header
	InitOKResponse(headersKeyValue ...string) io.Writer
	RequestCtx() context.Context
	ReplyError(statusCode int, args ...any)
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

// type IBLOBMessage interface {
// 	pipeline.IWorkpiece
// 	AppQName() appdef.AppQName
// 	WSID() istructs.WSID
// 	Header() http.Header
// 	InitOKResponse(headersKeyValue ...string) io.Writer
// 	Reader() io.ReadCloser
// 	RequestCtx() context.Context
// 	URLQueryValues() url.Values
// 	IsRead() bool // false -> write
// 	ReplyError(statusCode int, args ...any)
// 	ExistingBLOBIDOrSUUID() string
// }

type BLOBServiceChannel iprocbus.ServiceChannel

type BLOBServiceChannelGroupIdx uint

type NumBLOBProcessors uint
