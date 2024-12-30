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
	ibus "github.com/voedger/voedger/staging/src/github.com/untillpro/airs-ibus"
)

type BLOBOperation int

const (
	BLOBOperation_Null BLOBOperation = iota
	BLOBOperation_Read_Persistent
	BLOBOperation_Read_Temporary
	BLOBOperation_Write_Persistent_Single
	BLOBOperation_Write_Persistent_Multipart
	BLOBOperation_Write_Temporary_Single
	BLOBOperation_Write_Temporary_Multipart
)

type blobMessage_base struct {
	blobOperation   BLOBOperation
	req             *http.Request
	resp            http.ResponseWriter
	doneChan        chan struct{}
	wsid            istructs.WSID
	appQName        appdef.AppQName
	header          map[string][]string
	wLimiterFactory func() iblobstorage.WLimiterType
	sender          ibus.ISender

	duration iblobstorage.DurationType // used on write temporary
	boundary string                    // used on write multipart
	suuid    iblobstorage.SUUID        // used on read temporary
	blobid   istructs.RecordID         // used on read persistent
}

type WLimiterFactory func() iblobstorage.WLimiterType

type IBLOBMessage interface {
	pipeline.IWorkpiece
	AppQName() appdef.AppQName
	WSID() istructs.WSID
	Header() http.Header
	Sender() ibus.ISender
	InitOKResponse(headersKeyValue ...string) io.Writer
	Reader() io.ReadCloser
	RequestCtx() context.Context
	URL() *url.URL
	IsRead() bool // false -> write
}

type ServiceFactory func(serviceChannel iprocbus.ServiceChannel) pipeline.IService
