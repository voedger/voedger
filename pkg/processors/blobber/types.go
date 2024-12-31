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
	blobMessage           IBLOBMessage
	duration              iblobstorage.DurationType
	nameQuery             []string
	mimeTypeQuery         []string
	ttl                   string
	descr                 iblobstorage.DescrType
	mediaType             string
	boundary              string
	contentType           string
	existingBLOBIDOrSUUID string
	newBLOBID             istructs.RecordID
	doneCh                chan (interface{})
	blobState             iblobstorage.BLOBState
	blobKey               iblobstorage.IBLOBKey
	writer                io.Writer
	reader                io.Reader
	registerFuncName      string
}

type implIBLOBMessage struct {
	appQName         appdef.AppQName
	wsid             istructs.WSID
	header           http.Header
	requestCtx       context.Context
	urlQueryValues   url.Values
	okResponseIniter func(headersKeyValue ...string) io.Writer
	reader           io.ReadCloser
	errorResponder   ErrorResponder
	done             chan interface{}
}

func NewIBLOBMessage(appQName appdef.AppQName, wsid istructs.WSID, header http.Header, requestCtx context.Context,
	urlQueryValues url.Values, okResponseIniter func(headersKeyValue ...string) io.Writer, reader io.ReadCloser, errorResponder ErrorResponder) (msg IBLOBMessage, doneAwaiter func()) {
	msgImpl := &implIBLOBMessage{
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
	return msgImpl, func() {
	}
}

func (m *implIBLOBMessage) AppQName() appdef.AppQName {
	return m.appQName
}

func (m *implIBLOBMessage) WSID() istructs.WSID {
	return m.wsid
}

func (m *implIBLOBMessage) Header() http.Header {
	return m.header
}

func (m *implIBLOBMessage) RequestCtx() context.Context {
	return m.requestCtx
}

func (m *implIBLOBMessage) URLQueryValues() url.Values {
	return m.urlQueryValues
}

func (m *implIBLOBMessage) InitOKResponse(headersKeyValue ...string) io.Writer {
	return m.okResponseIniter(headersKeyValue...)
}

func (m *implIBLOBMessage) Reader() io.ReadCloser {
	return m.reader
}

func (m *implIBLOBMessage) IsRead() bool {
	return m.reader != nil
}

func (m *implIBLOBMessage) ReplyError(statusCode int, args ...any) {
	m.errorResponder(statusCode, args)
}

func (m *implIBLOBMessage) Release() {
	close(m.done)
}

type WLimiterFactory func() iblobstorage.WLimiterType

type IBLOBMessage interface {
	pipeline.IWorkpiece
	AppQName() appdef.AppQName
	WSID() istructs.WSID
	Header() http.Header
	InitOKResponse(headersKeyValue ...string) io.Writer
	Reader() io.ReadCloser
	RequestCtx() context.Context
	URLQueryValues() url.Values
	IsRead() bool // false -> write
	ReplyError(statusCode int, args ...any)
}

type BLOBServiceChannel iprocbus.ServiceChannel

type BLOBServiceChannelGroupIdx uint

type NumBLOBProcessors uint
