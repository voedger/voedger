/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package blobprocessor

import (
	"context"
	"net/http"

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
	// base
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
	AppQName() appdef.AppQName
	WSID() istructs.WSID
	BLOBOperation() BLOBOperation
	Header() map[string][]string
	Sender() ibus.ISender
	Resp() http.ResponseWriter
	RequestCtx() context.Context
	WLimiterFactory() WLimiterFactory
	Duration() iblobstorage.DurationType // used on write temporary
	SUUID() iblobstorage.SUUID           // used on read temporary
	Boundary() string                    // used on write multipart
	BLOBID() istructs.RecordID           // used on read persistent
}

type ServiceFactory func(serviceChannel iprocbus.ServiceChannel) pipeline.IService
