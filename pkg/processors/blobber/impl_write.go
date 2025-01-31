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

	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/coreutils/federation"
	"github.com/voedger/voedger/pkg/coreutils/utils"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/iblobstorage"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/pipeline"
)

func getRegisterFuncName(ctx context.Context, work pipeline.IWorkpiece) (err error) {
	bw := work.(*blobWorkpiece)
	if bw.isPersistent() {
		bw.registerFuncName = "c.sys.UploadBLOBHelper"
	} else {
		registerFuncName, ok := durationToRegisterFuncs[bw.duration]
		if !ok {
			// notest
			return coreutils.NewHTTPErrorf(http.StatusBadRequest, "unsupported blob duration value: ", bw.duration)
		}
		bw.registerFuncName = registerFuncName
	}
	return nil
}

func getBLOBKeyWrite(ctx context.Context, work pipeline.IWorkpiece) (err error) {
	bw := work.(*blobWorkpiece)
	if bw.isPersistent() {
		bw.blobKey = &iblobstorage.PersistentBLOBKeyType{
			ClusterAppID: istructs.ClusterAppID_sys_blobber,
			WSID:         bw.blobMessageWrite.wsid,
			BlobID:       bw.newBLOBID,
		}
	} else {
		// temp
		bw.blobKey = &iblobstorage.TempBLOBKeyType{
			ClusterAppID: istructs.ClusterAppID_sys_blobber,
			WSID:         bw.blobMessageWrite.wsid,
			SUUID:        bw.newSUUID,
		}
	}
	return nil
}

func provideWriteBLOB(blobStorage iblobstorage.IBLOBStorage, wLimiterFactory WLimiterFactory) func(ctx context.Context, work pipeline.IWorkpiece) (err error) {
	return func(ctx context.Context, work pipeline.IWorkpiece) (err error) {
		bw := work.(*blobWorkpiece)
		wLimiter := wLimiterFactory()
		if bw.isPersistent() {
			key := (bw.blobKey).(*iblobstorage.PersistentBLOBKeyType)
			err = blobStorage.WriteBLOB(bw.blobMessageWrite.requestCtx, *key, bw.descr, bw.blobMessageWrite.reader, wLimiter)
		} else {
			key := (bw.blobKey).(*iblobstorage.TempBLOBKeyType)
			err = blobStorage.WriteTempBLOB(ctx, *key, bw.descr, bw.blobMessageWrite.reader, wLimiter, bw.duration)
		}
		if errors.Is(err, iblobstorage.ErrBLOBSizeQuotaExceeded) {
			return coreutils.NewHTTPError(http.StatusForbidden, err)
		}
		return err
	}
}

func setBLOBStatusCompleted(ctx context.Context, work pipeline.IWorkpiece) (err error) {
	bw := work.(*blobWorkpiece)
	if !bw.isPersistent() {
		// do not account statuses for temp blobs
		return nil
	}
	// set WDoc<sys.BLOB>.status = BLOBStatus_Completed
	req := bus.Request{
		Method:   http.MethodPost,
		WSID:     bw.blobMessageWrite.wsid,
		AppQName: bw.blobMessageWrite.appQName,
		Resource: "c.sys.CUD",
		Body:     []byte(fmt.Sprintf(`{"cuds":[{"sys.ID": %d,"fields":{"status":%d}}]}`, bw.newBLOBID, iblobstorage.BLOBStatus_Completed)),
		Header:   bw.blobMessageWrite.header,
		Host:     coreutils.Localhost,
	}
	cudWDocBLOBUpdateMeta, cudWDocBLOBUpdateResp, err := bus.GetCommandResponse(bw.blobMessageWrite.requestCtx, bw.blobMessageWrite.requestSender, req)
	if err != nil {
		return fmt.Errorf("failed to exec c.sys.CUD: %w", err)
	}
	if cudWDocBLOBUpdateMeta.StatusCode != http.StatusOK {
		return coreutils.NewHTTPErrorf(cudWDocBLOBUpdateMeta.StatusCode, "c.sys.CUD returned error: ", cudWDocBLOBUpdateResp.SysError.Message)
	}
	return nil
}

func registerBLOB(ctx context.Context, work pipeline.IWorkpiece) (err error) {
	bw := work.(*blobWorkpiece)
	req := bus.Request{
		Method:   http.MethodPost,
		WSID:     bw.blobMessageWrite.wsid,
		AppQName: bw.blobMessageWrite.appQName,
		Resource: bw.registerFuncName,
		Header:   bw.blobMessageWrite.header,
		Body:     []byte(`{}`),
		Host:     coreutils.Localhost,
	}
	blobHelperMeta, blobHelperResp, err := bus.GetCommandResponse(bw.blobMessageWrite.requestCtx, bw.blobMessageWrite.requestSender, req)
	if err != nil {
		return fmt.Errorf("failed to exec q.sys.DownloadBLOBAuthnz: %w", err)
	}
	if blobHelperMeta.StatusCode != http.StatusOK {
		return coreutils.NewHTTPErrorf(blobHelperMeta.StatusCode, "q.sys.DownloadBLOBAuthnz returned error: "+blobHelperResp.SysError.Data)
	}
	if bw.isPersistent() {
		bw.newBLOBID = blobHelperResp.NewIDs["1"]
	} else {
		bw.newSUUID = iblobstorage.NewSUUID()
	}
	return nil
}

func getBLOBMessageWrite(_ context.Context, work pipeline.IWorkpiece) error {
	bw := work.(*blobWorkpiece)
	bw.blobMessageWrite = bw.blobMessage.(*implIBLOBMessage_Write)
	return nil
}

func parseQueryParams(_ context.Context, work pipeline.IWorkpiece) error {
	bw := work.(*blobWorkpiece)
	bw.nameQuery = bw.blobMessageWrite.urlQueryValues["name"]
	bw.mimeTypeQuery = bw.blobMessageWrite.urlQueryValues["mimeType"]
	if len(bw.blobMessageWrite.urlQueryValues["ttl"]) > 0 {
		bw.ttl = bw.blobMessageWrite.urlQueryValues["ttl"][0]
	}
	return nil
}

func parseMediaType(_ context.Context, work pipeline.IWorkpiece) error {
	bw := work.(*blobWorkpiece)
	bw.contentType = bw.blobMessageWrite.header[coreutils.ContentType]
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

func (b *sendWriteResult) DoSync(_ context.Context, work pipeline.IWorkpiece) (err error) {
	bw := work.(*blobWorkpiece)
	if bw.resultErr == nil {
		if logger.IsVerbose() {
			blobIDStr := fmt.Sprint(bw.newBLOBID)
			if len(blobIDStr) == 0 {
				blobIDStr = string(bw.newSUUID)
			}
			logger.Verbose("blob write success:", bw.nameQuery, ":", bw.newBLOBID)
		}
		writer := bw.blobMessageWrite.okResponseIniter(coreutils.ContentType, "text/plain")
		if bw.isPersistent() {
			_, _ = writer.Write([]byte(utils.UintToString(bw.newBLOBID)))
		} else {
			_, _ = writer.Write([]byte(bw.newSUUID))
		}
		return nil
	}
	var sysError coreutils.SysError
	errors.As(bw.resultErr, &sysError)
	if logger.IsVerbose() {
		logger.Verbose("blob write error:", sysError.HTTPStatus, ":", sysError.Message)
	}
	bw.blobMessageWrite.errorResponder(sysError.HTTPStatus, sysError.Message)
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
