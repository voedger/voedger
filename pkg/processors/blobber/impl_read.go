/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package blobprocessor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/coreutils/utils"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/iblobstorage"
	"github.com/voedger/voedger/pkg/iblobstoragestg"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/pipeline"
	ibus "github.com/voedger/voedger/staging/src/github.com/untillpro/airs-ibus"
)

// TODO: blobStorage - кэшированный или нет? Елси нет, то не надо QueryState выделять в отдельный оператор, а то 2 раза в хранилище будем лазить
// TODO: blobMessage.Release() должен закрывать io.ReadCloser

// func getBLOBKey(ctx context.Context, work pipeline.IWorkpiece) (err error) {
// 	bm := work.(*blobWorkpiece)
// 	if bm.isPersistent() {
// 		if
// 		blobID, err := strconv.ParseUint(bm.existingBLOBIDOrSUUID, utils.DecimalBase, utils.BitSize64)
// 		if err != nil {
// 			// validated already by router
// 			// notest
// 			return err
// 		}
// 		bm.blobKey = &iblobstorage.PersistentBLOBKeyType{
// 			ClusterAppID: istructs.ClusterAppID_sys_blobber,
// 			WSID:         bm.blobMessage.WSID(),
// 			BlobID:       istructs.RecordID(blobID),
// 		}
// 	} else {
// 		bm.blobKey = &iblobstorage.TempBLOBKeyType{
// 			ClusterAppID: istructs.ClusterAppID_sys_blobber,
// 			WSID:         bm.blobMessage.WSID(),
// 			SUUID:        iblobstorage.SUUID(bm.existingBLOBIDOrSUUID),
// 		}
// 	}
// 	return nil
// }

func getBLOBKey(ctx context.Context, work pipeline.IWorkpiece) (err error) {
	bm := work.(*blobWorkpiece)
	if bm.isPersistent() {
		var blobID istructs.RecordID
		if bm.blobMessage.IsRead() {
			blobIDUint, err := strconv.ParseUint(bm.existingBLOBIDOrSUUID, utils.DecimalBase, utils.BitSize64)
			if err != nil {
				// validated already by router
				// notest
				return err
			}
			blobID = istructs.RecordID(blobIDUint)
		} else {
			blobID = bm.newBLOBID
		}
		bm.blobKey = &iblobstorage.PersistentBLOBKeyType{
			ClusterAppID: istructs.ClusterAppID_sys_blobber,
			WSID:         bm.blobMessage.WSID(),
			BlobID:       blobID,
		}
	} else {
		var suuid iblobstorage.SUUID
		if bm.blobMessage.IsRead() {
			suuid = iblobstorage.SUUID(bm.existingBLOBIDOrSUUID)
		} else {
			suuid = bm.newSUUID
		}
		bm.blobKey = &iblobstorage.TempBLOBKeyType{
			ClusterAppID: istructs.ClusterAppID_sys_blobber,
			WSID:         bm.blobMessage.WSID(),
			SUUID:        suuid,
		}
	}
	return nil
}

func readBLOBID(ctx context.Context, work pipeline.IWorkpiece) (err error) {
	bw := work.(*blobWorkpiece)
	bodyBytes, err := io.ReadAll(bw.blobMessage.Reader())
	if err != nil {
		// notest
		return fmt.Errorf("failed to read request body: %w", err)
	}
	bw.existingBLOBIDOrSUUID = string(bodyBytes)
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

func provideDownloadBLOBHelper(bus ibus.IBus, busTimeout time.Duration) func(ctx context.Context, work pipeline.IWorkpiece) (err error) {
	return func(ctx context.Context, work pipeline.IWorkpiece) (err error) {
		bm := work.(*blobWorkpiece)
		// request to VVM to check the principalToken
		req := ibus.Request{
			Method:   ibus.HTTPMethodPOST,
			WSID:     bm.blobMessage.WSID(),
			AppQName: bm.blobMessage.AppQName().String(),
			Resource: "q.sys.DownloadBLOBAuthnz",
			Header:   bm.blobMessage.Header(),
			Body:     []byte(`{}`),
			Host:     coreutils.Localhost,
		}
		blobHelperResp, _, _, err := bus.SendRequest2(bm.blobMessage.RequestCtx(), req, busTimeout)
		if err != nil {
			return coreutils.NewHTTPErrorf(http.StatusInternalServerError, "failed to exec q.sys.DownloadBLOBAuthnz: ", err)
		}
		if blobHelperResp.StatusCode != http.StatusOK {
			return coreutils.NewHTTPErrorf(blobHelperResp.StatusCode, "q.sys.DownloadBLOBAuthnz returned error: "+string(blobHelperResp.Data))
		}
		return nil
	}
}

func provideReadBLOB(blobStorage iblobstorage.IBLOBStorage) func(ctx context.Context, work pipeline.IWorkpiece) (err error) {
	return func(ctx context.Context, work pipeline.IWorkpiece) (err error) {
		bm := work.(*blobWorkpiece)
		err = blobStorage.ReadBLOB(bm.blobMessage.RequestCtx(), bm.blobKey, nil, bm.writer, iblobstoragestg.RLimiter_Null)
		if err != nil {
			logger.Error(fmt.Sprintf("failed to read BLOB: id %s, appQName %s, wsid %d: %s", bm.blobKey.ID(), bm.blobMessage.AppQName(),
				bm.blobMessage.WSID(), err.Error()))
			if errors.Is(err, iblobstorage.ErrBLOBNotFound) {
				return coreutils.NewHTTPError(http.StatusNotFound, err)
			}
			return coreutils.NewHTTPError(http.StatusInternalServerError, err)
		}
		return nil
	}
}
