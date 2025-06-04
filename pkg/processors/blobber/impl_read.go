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
	"github.com/voedger/voedger/pkg/processors"
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
		coreutils.ContentType, bw.blobState.Descr.ContentType,
		coreutils.BlobName, bw.blobState.Descr.Name,
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
	if !bw.blobMessageRead.isAPIv2 {
		return nil
	}
	req := bus.Request{
		Method:   http.MethodGet,
		WSID:     bw.blobMessageRead.wsid,
		AppQName: bw.blobMessageRead.appQName,
		Header:   bw.blobMessageRead.header,
		Body:     []byte(`{}`),
		Host:     coreutils.Localhost,
		APIPath:  int(processors.APIPath_Queries),
		IsAPIV2:  true,
		QName:    downloadPersistentBLOBFuncQName,
	}
	_, err = bus.ReadQueryResponse(bw.blobMessageRead.requestCtx, bw.blobMessageRead.requestSender, req)
	if err != nil {
		return fmt.Errorf("failed to exec q.sys.DownloadBLOBAuthnz: %w", err)
	}
	return nil
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

func getBLOBIDFromOwner(_ context.Context, work pipeline.IWorkpiece) (err error) {
	bw := work.(*blobWorkpiece)
	if !bw.blobMessageRead.isAPIv2 || !bw.isPersistent() {
		// temp blob in APIv2 -> skip, suuid is already known
		return nil
	}
	req := bus.Request{
		Method:   http.MethodGet,
		WSID:     bw.blobMessageRead.wsid,
		AppQName: bw.blobMessageRead.appQName,
		Header:   bw.blobMessageRead.header,
		APIPath:  int(processors.APIPath_Docs),
		DocID:    istructs.IDType(bw.blobMessageRead.ownerID),
		QName:    bw.blobMessageRead.ownerRecord,
		Host:     coreutils.Localhost,
		IsAPIV2:  true,
		Query: map[string]string{
			"keys": bw.blobMessageRead.ownerRecordField,
		},
	}
	resp, err := bus.ReadQueryResponse(bw.blobMessageRead.requestCtx, bw.blobMessageRead.requestSender, req)
	if err != nil {
		// notest
		return fmt.Errorf("failed to read BLOBID from owner: %w", err)
	}
	if len(resp) > 1 {
		// notest
		return coreutils.NewHTTPErrorf(http.StatusInternalServerError, fmt.Errorf("unexpected result reading BLOBID from owner: multiple responses received"))
	}
	ownerFieldValue, ok := resp[0][bw.blobMessageRead.ownerRecordField]
	if !ok {
		return coreutils.NewHTTPErrorf(http.StatusBadRequest, fmt.Errorf("no value for owner field %s in blob owner doc %s", bw.blobMessageRead.ownerRecordField, bw.blobMessageRead.ownerRecord))
	}
	blobID, ok := ownerFieldValue.(istructs.RecordID)
	if !ok {
		return coreutils.NewHTTPErrorf(http.StatusBadRequest, fmt.Errorf("owner field %s.%s is not of blob type", bw.blobMessageRead.ownerRecord, bw.blobMessageRead.ownerRecordField))
	}
	bw.blobMessageRead.existingBLOBIDOrSUUID = utils.UintToString(blobID)
	return nil
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
		bw.blobMessageRead.errorResponder(sysError.HTTPStatus, sysError)
		return nil
	}
	return bw.resultErr
}
