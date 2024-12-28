/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package blobprocessor

import (
	"context"
	"errors"
	"fmt"
	"mime"
	"time"

	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/coreutils/federation"
	"github.com/voedger/voedger/pkg/iblobstorage"
	"github.com/voedger/voedger/pkg/iprocbus"
	"github.com/voedger/voedger/pkg/pipeline"
	ibus "github.com/voedger/voedger/staging/src/github.com/untillpro/airs-ibus"
)

func ProvideService(serviceChannel iprocbus.ServiceChannel, blobStorage iblobstorage.IBLOBStorage,
	ibus ibus.IBus, busTimeout time.Duration) pipeline.IService {
	return pipeline.NewService(func(ctx context.Context) {
		for ctx.Err() == nil {
			select {
			case workIntf := <-serviceChannel:
				blobMessage := workIntf.(IBLOBMessage)
				switch blobMessage.BLOBOperation() {
				case BLOBOperation_Read_Persistent, BLOBOperation_Read_Temporary:
					readBLOB(blobMessage, blobStorage, ibus, busTimeout)
				case BLOBOperation_Write_Persistent_Single:
					writeBLOB(blobMessage, blobStorage, busTimeout, ibus)
				}
			case <-ctx.Done():
			}
		}
	})
}

type blobWorkpiece struct {
	blobMessage   IBLOBMessage
	opKind        BLOBOperation
	duration      iblobstorage.DurationType
	nameQuery     []string
	mimeTypeQuery []string
	ttl           string
	descr         iblobstorage.DescrType
	mediaType     string
	boundary      string
	contentType   string
}

func (b *blobWorkpiece) Release() {}

func parseQueryParams(_ context.Context, work pipeline.IWorkpiece) error {
	bw := work.(*blobWorkpiece)
	values := bw.blobMessage.URL().Query()
	bw.nameQuery = values["name"]
	bw.mimeTypeQuery = values["mimeType"]
	if len(values["ttl"]) > 0 {
		bw.ttl = values["ttl"][0]
	}
	return nil
}

func parseMediaType(_ context.Context, work pipeline.IWorkpiece) error {
	bw := work.(*blobWorkpiece)
	bw.contentType = bw.blobMessage.Header().Get(coreutils.ContentType)
	mediaType, params, err := mime.ParseMediaType(bw.contentType)
	if err != nil {
		return fmt.Errorf("failed ot parse Content-Type header: %w", err)
	}
	bw.mediaType = mediaType
	bw.boundary = params["boundary"]
	return nil
}

func validateParams(_ context.Context, work pipeline.IWorkpiece) error {
	bw := work.(*blobWorkpiece)

	if (len(bw.nameQuery) > 0 && len(bw.mimeTypeQuery) == 0) || (len(bw.nameQuery) == 0 && len(bw.mimeTypeQuery) > 0) {
		return errors.New("both name and mimeType query params must be specified")
	}

	isSingleBLOB := len(bw.nameQuery) > 0 && len(bw.mimeTypeQuery) > 0

	if len(bw.ttl) > 0 {
		duration, ttlSupported := federation.TemporaryBLOB_URLTTLToDurationLs[bw.ttl]
		if !ttlSupported {
			return errors.New(`"1d" is only supported for now for temporary blob ttl`)
		}
		bw.duration = duration
	}

	if isSingleBLOB {
		if bw.contentType == coreutils.MultipartFormData {
			return fmt.Errorf(`name+mimeType query params and "%s" Content-Type header are mutual exclusive`, coreutils.MultipartFormData)
		}
		bw.descr.Name = bw.nameQuery[0]
		bw.descr.MimeType = bw.mimeTypeQuery[0]
		return nil
	}

	// not a single BLOB
	if len(bw.contentType) == 0 {
		return errors.New(`neither "name"+"mimeType" query params nor Content-Type header is not provided`)
	}

	if bw.mediaType != coreutils.MultipartFormData {
		return errors.New("name+mimeType query params are not provided -> Content-Type must be mutipart/form-data but actual is " + bw.contentType)
	}

	if len(bw.boundary) == 0 {
		return fmt.Errorf("boundary of %s is not specified", coreutils.MultipartFormData)
	}
	pipeline.ISwitch
	return nil
}

func providePipeline(vvmCtx context.Context) pipeline.ISyncPipeline {
	return pipeline.NewSyncPipeline(vvmCtx, "blob processor",
		pipeline.WireFunc("parseQueryParams", parseQueryParams),
		pipeline.WireFunc("parseMediaType", parseMediaType),
		pipeline.WireFunc("validateParams", validateParams),
		pipeline.WireFunc("checkBadRequest", )
		pipeline.WireSyncOperator("switch", pipeline.SwitchOperator(pipeline.sw)),
	)
}
