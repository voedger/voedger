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
	"github.com/voedger/voedger/pkg/coreutils/utils"
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
					blobMessage: workIntf.(iBLOBMessage_Base),
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

func getBLOBMessageRead(_ context.Context, work pipeline.IWorkpiece) error {
	bw := work.(*blobWorkpiece)
	bw.blobMessageRead = bw.blobMessage.(IBLOBMessage_Read)
	return nil
}

func getBLOBMessageWrite(_ context.Context, work pipeline.IWorkpiece) error {
	bw := work.(*blobWorkpiece)
	bw.blobMessageWrite = bw.blobMessage.(IBLOBMessage_Write)
	return nil
}

func parseQueryParams(_ context.Context, work pipeline.IWorkpiece) error {
	bw := work.(*blobWorkpiece)
	bw.nameQuery = bw.blobMessageWrite.URLQueryValues()["name"]
	bw.mimeTypeQuery = bw.blobMessageWrite.URLQueryValues()["mimeType"]
	if len(bw.blobMessageWrite.URLQueryValues()["ttl"]) > 0 {
		bw.ttl = bw.blobMessageWrite.URLQueryValues()["ttl"][0]
	}
	return nil
}

func parseMediaType(_ context.Context, work pipeline.IWorkpiece) error {
	bw := work.(*blobWorkpiece)
	bw.contentType = bw.blobMessage.Header().Get(coreutils.ContentType)
	if len(bw.contentType) == 0 {
		return nil
	}
	mediaType, params, err := mime.ParseMediaType(bw.contentType)
	if err != nil {
		return fmt.Errorf("failed to parse Content-Type header: %w", err)
	}
	bw.mediaType = mediaType
	bw.boundary = params["boundary"]
	return nil
}

func validateQueryParams(_ context.Context, work pipeline.IWorkpiece) error {
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

type sendWriteResult struct {
	pipeline.NOOP
}

type catchReadError struct {
	pipeline.NOOP
}

func (b *catchReadError) OnErr(err error, work interface{}, _ pipeline.IWorkpieceContext) (newErr error) {
	bw := work.(*blobWorkpiece)
	bw.resultErr = coreutils.WrapSysError(err, http.StatusInternalServerError)
	return nil
}

func (b *catchReadError) DoSync(_ context.Context, work pipeline.IWorkpiece) (err error) {
	bw := work.(*blobWorkpiece)
	var sysError coreutils.SysError
	if errors.As(bw.resultErr, &sysError) {
		bw.blobMessage.ReplyError(sysError.HTTPStatus, sysError.Message)
	}
	return nil
}

func (b *sendWriteResult) DoSync(_ context.Context, work pipeline.IWorkpiece) (err error) {
	bw := work.(*blobWorkpiece)
	if bw.resultErr == nil {
		writer := bw.blobMessage.InitOKResponse(coreutils.ContentType, "text/plain")
		if bw.isPersistent() {
			_, _ = writer.Write([]byte(utils.UintToString(bw.newBLOBID)))
		} else {
			_, _ = writer.Write([]byte(bw.newSUUID))
		}
		return nil
	}
	var sysError coreutils.SysError
	errors.As(bw.resultErr, &sysError)
	bw.blobMessage.ReplyError(sysError.HTTPStatus, sysError.Message)
	return nil
}

func (b *sendWriteResult) OnErr(err error, work interface{}, _ pipeline.IWorkpieceContext) (newErr error) {
	bw := work.(*blobWorkpiece)
	bw.resultErr = coreutils.WrapSysError(err, http.StatusInternalServerError)
	return nil
}

func (b *badRequestWrapper) OnErr(err error, _ interface{}, _ pipeline.IWorkpieceContext) (newErr error) {
	return coreutils.WrapSysError(err, http.StatusBadRequest)
}

type blobOpSwitch struct {
}

func (b *blobOpSwitch) Switch(work interface{}) (branchName string, err error) {
	blobWorkpiece := work.(*blobWorkpiece)
	if _, ok := blobWorkpiece.blobMessage.(IBLOBMessage_Read); ok {
		return "readBLOB", nil
	}
	return "writeBLOB", nil
}

func providePipeline(vvmCtx context.Context, blobStorage iblobstorage.IBLOBStorage, bus ibus.IBus, busTimeout time.Duration,
	wLimiterFactory WLimiterFactory) pipeline.ISyncPipeline {
	return pipeline.NewSyncPipeline(vvmCtx, "blob processor",
		pipeline.WireSyncOperator("switch", pipeline.SwitchOperator(&blobOpSwitch{},
			pipeline.SwitchBranch("readBLOB", pipeline.NewSyncPipeline(vvmCtx, "readBLOB",
				pipeline.WireFunc("getBLOBMessageRead", getBLOBMessageRead),
				pipeline.WireFunc("downloadBLOBHelper", provideDownloadBLOBHelper(bus, busTimeout)),
				pipeline.WireFunc("getBLOBKeyRead", getBLOBKeyRead),
				pipeline.WireFunc("queryBLOBState", provideQueryAndCheckBLOBState(blobStorage)),
				pipeline.WireFunc("initResponse", initResponse),
				pipeline.WireFunc("readBLOB", provideReadBLOB(blobStorage)),
				pipeline.WireSyncOperator("catchReadError", &catchReadError{}),
			)),
			pipeline.SwitchBranch("writeBLOB", pipeline.NewSyncPipeline(vvmCtx, "writeBLOB",
				pipeline.WireFunc("getBLOBMessageWrite", getBLOBMessageWrite),
				pipeline.WireFunc("parseQueryParams", parseQueryParams),
				pipeline.WireFunc("parseMediaType", parseMediaType),
				pipeline.WireFunc("validateQueryParams", validateQueryParams),
				pipeline.WireFunc("getRegisterFuncName", getRegisterFuncName),
				pipeline.WireSyncOperator("wrapBadRequest", &badRequestWrapper{}),
				pipeline.WireFunc("registerBLOB", provideRegisterBLOB(bus, busTimeout)),
				pipeline.WireFunc("getBLOBKeyWrite", getBLOBKeyWrite),
				pipeline.WireFunc("writeBLOB", provideWriteBLOB(blobStorage, wLimiterFactory)),
				pipeline.WireFunc("setBLOBStatusCompleted", provideSetBLOBStatusCompleted(bus, busTimeout)),
				pipeline.WireSyncOperator("sendResult", &sendWriteResult{}),
			)),
		)),
	)
}

func (b *blobWorkpiece) isPersistent() bool {
	if b.blobMessageWrite != nil {
		return len(b.ttl) == 0
	}
	return len(b.blobMessageRead.ExistingBLOBIDOrSUUID()) <= temporaryBLOBIDLenTreshold
}
