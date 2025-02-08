/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package blobprocessor

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/coreutils/utils"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/iblobstorage"
	"github.com/voedger/voedger/pkg/iblobstoragestg"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/pipeline"
)

func getBLOBKeyRead(ctx context.Context, work pipeline.IWorkpiece) (err error) {
	bw := work.(*blobWorkpiece)
	if bw.isPersistent() {
		existingBLOBIDUint, err := strconv.ParseUint(bw.blobMessageRead.existingBLOBIDOrSUUID, utils.DecimalBase, utils.BitSize64)
		if err != nil {
			// validated already by router
			// notest
			return err
		}
		existingBLOBID := istructs.RecordID(existingBLOBIDUint)
		bw.blobKey = &iblobstorage.PersistentBLOBKeyType{
			ClusterAppID: istructs.ClusterAppID_sys_blobber,
			WSID:         bw.blobMessageRead.wsid,
			BlobID:       existingBLOBID,
		}
		return nil
	}

	// temp
	bw.blobKey = &iblobstorage.TempBLOBKeyType{
		ClusterAppID: istructs.ClusterAppID_sys_blobber,
		WSID:         bw.blobMessageRead.wsid,
		SUUID:        iblobstorage.SUUID(bw.blobMessageRead.existingBLOBIDOrSUUID),
	}
	return nil
}

func initResponse(ctx context.Context, work pipeline.IWorkpiece) (err error) {
	bw := work.(*blobWorkpiece)
	bw.writer = bw.blobMessageRead.okResponseIniter(
		coreutils.ContentType, bw.blobState.Descr.MimeType,
		"Content-Disposition", fmt.Sprintf(`attachment;filename="%s"`, bw.blobState.Descr.Name),
	)
	return nil
}

func provideQueryAndCheckBLOBState(blobStorage iblobstorage.IBLOBStorage) func(ctx context.Context, work pipeline.IWorkpiece) (err error) {
	return func(ctx context.Context, work pipeline.IWorkpiece) (err error) {
		bw := work.(*blobWorkpiece)
		bw.blobState, err = blobStorage.QueryBLOBState(bw.blobMessageRead.requestCtx, bw.blobKey)
		if err != nil {
			if errors.Is(err, iblobstorage.ErrBLOBNotFound) {
				return coreutils.NewHTTPError(http.StatusNotFound, err)
			}
			return err
		}
		if bw.blobState.Status != iblobstorage.BLOBStatus_Completed {
			return errors.New("blob is not completed")
		}
		if len(bw.blobState.Error) > 0 {
			return errors.New(bw.blobState.Error)
		}
		return nil
	}
}

func downloadBLOBHelper(ctx context.Context, work pipeline.IWorkpiece) (err error) {
	bw := work.(*blobWorkpiece)
	req := bus.Request{
		Method:   http.MethodPost,
		WSID:     bw.blobMessageRead.wsid,
		AppQName: bw.blobMessageRead.appQName,
		Resource: "q.sys.DownloadBLOBAuthnz",
		Header:   bw.blobMessageRead.header,
		Body:     []byte(`{}`),
		Host:     coreutils.Localhost,
	}
	respCh, _, respErr, err := bw.blobMessageRead.requestSender.SendRequest(bw.blobMessageRead.requestCtx, req)
	if err != nil {
		return fmt.Errorf("failed to exec q.sys.DownloadBLOBAuthnz: %w", err)
	}
	for range respCh {
		// notest
		panic("unexpeced result of q.sys.DownloadBLOBAuthnz")
	}
	return *respErr
}

func provideReadBLOB(blobStorage iblobstorage.IBLOBStorage) func(ctx context.Context, work pipeline.IWorkpiece) (err error) {
	return func(ctx context.Context, work pipeline.IWorkpiece) (err error) {
		bw := work.(*blobWorkpiece)
		err = blobStorage.ReadBLOB(bw.blobMessageRead.requestCtx, bw.blobKey, nil, bw.writer, iblobstoragestg.RLimiter_Null)
		if err != nil {
			logger.Error(fmt.Sprintf("failed to read BLOB: id %s, appQName %s, wsid %d: %s", bw.blobKey.ID(), bw.blobMessageRead.appQName,
				bw.blobMessageRead.wsid, err.Error()))
		}
		return err
	}
}

func getBLOBMessageRead(_ context.Context, work pipeline.IWorkpiece) error {
	bw := work.(*blobWorkpiece)
	bw.blobMessageRead = bw.blobMessage.(*implIBLOBMessage_Read)
	return nil
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
		if logger.IsVerbose() {
			logger.Verbose("blob read error:", sysError.HTTPStatus, ":", sysError.Message)
		}
		bw.blobMessageRead.errorResponder(sysError.HTTPStatus, sysError.Message)
		return nil
	}
	return bw.resultErr
}
