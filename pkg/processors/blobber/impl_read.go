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

	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/coreutils/bus"
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
		existingBLOBIDUint, err := strconv.ParseUint(bw.blobMessageRead.ExistingBLOBIDOrSUUID(), utils.DecimalBase, utils.BitSize64)
		if err != nil {
			// validated already by router
			// notest
			return err
		}
		existingBLOBID := istructs.RecordID(existingBLOBIDUint)
		bw.blobKey = &iblobstorage.PersistentBLOBKeyType{
			ClusterAppID: istructs.ClusterAppID_sys_blobber,
			WSID:         bw.blobMessage.WSID(),
			BlobID:       existingBLOBID,
		}
		return nil
	}

	// temp
	bw.blobKey = &iblobstorage.TempBLOBKeyType{
		ClusterAppID: istructs.ClusterAppID_sys_blobber,
		WSID:         bw.blobMessage.WSID(),
		SUUID:        iblobstorage.SUUID(bw.blobMessageRead.ExistingBLOBIDOrSUUID()),
	}
	return nil
}

func getBLOBKeyWrite(ctx context.Context, work pipeline.IWorkpiece) (err error) {
	bw := work.(*blobWorkpiece)
	if bw.isPersistent() {
		bw.blobKey = &iblobstorage.PersistentBLOBKeyType{
			ClusterAppID: istructs.ClusterAppID_sys_blobber,
			WSID:         bw.blobMessage.WSID(),
			BlobID:       bw.newBLOBID,
		}
		return nil
	}

	// temp
	bw.blobKey = &iblobstorage.TempBLOBKeyType{
		ClusterAppID: istructs.ClusterAppID_sys_blobber,
		WSID:         bw.blobMessage.WSID(),
		SUUID:        bw.newSUUID,
	}
	return nil
}

func initResponse(ctx context.Context, work pipeline.IWorkpiece) (err error) {
	bm := work.(*blobWorkpiece)
	bm.writer = bm.blobMessage.InitOKResponse(
		coreutils.ContentType, bm.blobState.Descr.MimeType,
		"Content-Disposition", fmt.Sprintf(`attachment;filename="%s"`, bm.blobState.Descr.Name),
	)
	return nil
}

func provideQueryAndCheckBLOBState(blobStorage iblobstorage.IBLOBStorage) func(ctx context.Context, work pipeline.IWorkpiece) (err error) {
	return func(ctx context.Context, work pipeline.IWorkpiece) (err error) {
		bm := work.(*blobWorkpiece)
		bm.blobState, err = blobStorage.QueryBLOBState(bm.blobMessage.RequestCtx(), bm.blobKey)
		if err != nil {
			if errors.Is(err, iblobstorage.ErrBLOBNotFound) {
				return coreutils.NewHTTPError(http.StatusNotFound, err)
			}
			return err
		}
		if bm.blobState.Status != iblobstorage.BLOBStatus_Completed {
			return errors.New("blob is not completed")
		}
		if len(bm.blobState.Error) > 0 {
			return errors.New(bm.blobState.Error)
		}
		return nil
	}
}

func downloadBLOBHelper(ctx context.Context, work pipeline.IWorkpiece) (err error) {
	bw := work.(*blobWorkpiece)
	req := bus.Request{
		Method:   http.MethodPost,
		WSID:     bw.blobMessage.WSID(),
		AppQName: bw.blobMessage.AppQName().String(),
		Resource: "q.sys.DownloadBLOBAuthnz",
		Header:   bw.blobMessage.Header(),
		Body:     []byte(`{}`),
		Host:     coreutils.Localhost,
	}
	respCh, _, respErr, err := bw.blobMessage.RequestSender().SendRequest(bw.blobMessage.RequestCtx(), req)
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
		bm := work.(*blobWorkpiece)
		err = blobStorage.ReadBLOB(bm.blobMessage.RequestCtx(), bm.blobKey, nil, bm.writer, iblobstoragestg.RLimiter_Null)
		if err != nil {
			logger.Error(fmt.Sprintf("failed to read BLOB: id %s, appQName %s, wsid %d: %s", bm.blobKey.ID(), bm.blobMessage.AppQName(),
				bm.blobMessage.WSID(), err.Error()))
			return coreutils.NewHTTPError(http.StatusInternalServerError, err)
		}
		return nil
	}
}

func getBLOBMessageRead(_ context.Context, work pipeline.IWorkpiece) error {
	bw := work.(*blobWorkpiece)
	bw.blobMessageRead = bw.blobMessage.(IBLOBMessage_Read)
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
		bw.blobMessage.ReplyError(sysError.HTTPStatus, sysError.Message)
	}
	return nil
}
