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
	"net/http"
	"time"

	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/coreutils/federation"
	"github.com/voedger/voedger/pkg/iblobstorage"
	"github.com/voedger/voedger/pkg/pipeline"
	ibus "github.com/voedger/voedger/staging/src/github.com/untillpro/airs-ibus"
)

func ProvideService(serviceChannel BLOBServiceChannel, blobStorage iblobstorage.IBLOBStorage,
	ibus ibus.IBus, busTimeout time.Duration, wLimiterFactory WLimiterFactory) pipeline.IService {
	return pipeline.NewService(func(vvmCtx context.Context) {
		pipeline := providePipeline(vvmCtx, blobStorage, ibus, busTimeout, wLimiterFactory)
		for vvmCtx.Err() == nil {
			select {
			case workIntf := <-serviceChannel:
				blobWorkpiece := &blobWorkpiece{
					blobMessage: workIntf.(IBLOBMessage),
				}
				if err := pipeline.SendSync(blobWorkpiece); err != nil {
					// notest
					panic(err)
				}
				blobWorkpiece.Release()
			case <-vvmCtx.Done():
			}
		}
		pipeline.Close()
	})
}

func (b *blobWorkpiece) Release() {
	b.blobMessage.Release()
}

func parseQueryParams(_ context.Context, work pipeline.IWorkpiece) error {
	bw := work.(*blobWorkpiece)
	bw.nameQuery = bw.blobMessage.URLQueryValues()["name"]
	bw.mimeTypeQuery = bw.blobMessage.URLQueryValues()["mimeType"]
	if len(bw.blobMessage.URLQueryValues()["ttl"]) > 0 {
		bw.ttl = bw.blobMessage.URLQueryValues()["ttl"][0]
	}
	return nil
}

func parseMediaType(_ context.Context, work pipeline.IWorkpiece) error {
	bw := work.(*blobWorkpiece)
	if len(bw.nameQuery) > 0 {
		return nil
	}
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
	return nil
}

type badRequestWrapper struct {
	pipeline.NOOP
}

func (b *badRequestWrapper) OnErr(err error, _ interface{}, _ pipeline.IWorkpieceContext) (newErr error) {
	return coreutils.WrapSysError(err, http.StatusBadRequest)
}

type blobOpSwitch struct {
}

func (b *blobOpSwitch) Switch(work interface{}) (branchName string, err error) {
	blobWorkpiece := work.(*blobWorkpiece)
	if blobWorkpiece.blobMessage.IsRead() {
		тут неправильно или перепутано
		return "readBLOB", nil
	}
	return "writeBLOB", nil
}

func providePipeline(vvmCtx context.Context, blobStorage iblobstorage.IBLOBStorage, bus ibus.IBus, busTimeout time.Duration,
	wLimiterFactory WLimiterFactory) pipeline.ISyncPipeline {
	return pipeline.NewSyncPipeline(vvmCtx, "blob processor",
		pipeline.WireFunc("parseQueryParams", parseQueryParams),
		pipeline.WireFunc("parseMediaType", parseMediaType),
		pipeline.WireFunc("validateParams", validateParams),
		pipeline.WireSyncOperator("wrapBadRequest", &badRequestWrapper{}),
		pipeline.WireSyncOperator("switch", pipeline.SwitchOperator(&blobOpSwitch{},
			pipeline.SwitchBranch("readBLOB", pipeline.NewSyncPipeline(vvmCtx, "readBLOB",
				pipeline.WireFunc("readBLOBID", readBLOBID),
				pipeline.WireFunc("downloadBLOBHelper", provideDownloadBLOBHelper(bus, busTimeout)),
				pipeline.WireFunc("getBLOBKey", getBLOBKey),
				pipeline.WireFunc("queryBLOBState", provideQueryAndCheckBLOBState(blobStorage)),
				pipeline.WireFunc("initResponse", initResponse),
				pipeline.WireFunc("readBLOB", provideReadBLOB(blobStorage)),
			)),
			pipeline.SwitchBranch("writeBLOB", pipeline.NewSyncPipeline(vvmCtx, "writeBLOB",
				pipeline.WireFunc("getRegisterFuncName", getRegisterFuncName),
				pipeline.WireFunc("registerBLOB", provideRegisterBLOB(bus, busTimeout)),
				pipeline.WireFunc("writeBLOB", provideWriteBLOB(blobStorage, wLimiterFactory)),
				pipeline.WireFunc("setBLOBStatusCompleted", provideSetBLOBStatusCompleted(bus, busTimeout)),
			)),
		)),
	)
}

func (b *blobWorkpiece) isPersistent() bool {
	return len(b.existingBLOBIDOrSUUID) <= temporaryBLOBIDLenTreshold
}
