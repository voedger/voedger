/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package blobprocessor

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/iblobstorage"
	"github.com/voedger/voedger/pkg/iblobstoragestg"
	"github.com/voedger/voedger/pkg/istructs"
	ibus "github.com/voedger/voedger/staging/src/github.com/untillpro/airs-ibus"
)

func readBLOB(bm IBLOBMessage, blobStorage iblobstorage.IBLOBStorage, bus ibus.IBus, busTimeout time.Duration) {
	// request to VVM to check the principalToken
	req := ibus.Request{
		Method:   ibus.HTTPMethodPOST,
		WSID:     bm.WSID(),
		AppQName: bm.AppQName().String(),
		Resource: "q.sys.DownloadBLOBAuthnz",
		Header:   bm.Header(),
		Body:     []byte(`{}`),
		Host:     coreutils.Localhost,
	}
	blobHelperResp, _, _, err := bus.SendRequest2(bm.RequestCtx(), req, busTimeout)
	if err != nil {
		coreutils.ReplyInternalServerError(bm.Sender(), "failed to exec q.sys.DownloadBLOBAuthnz", err)
		return
	}
	if blobHelperResp.StatusCode != http.StatusOK {
		coreutils.ReplyErrf(bm.Sender(), blobHelperResp.StatusCode, "q.sys.DownloadBLOBAuthnz returned error: "+string(blobHelperResp.Data))
		return
	}

	// read the BLOB
	var blobKey iblobstorage.IBLOBKey
	switch bm.BLOBOperation() {
	case BLOBOperation_Read_Persistent:
		blobKey = &iblobstorage.PersistentBLOBKeyType{
			ClusterAppID: istructs.ClusterAppID_sys_blobber,
			WSID:         bm.WSID(),
			BlobID:       bm.BLOBID(),
		}
	case BLOBOperation_Read_Temporary:
		blobKey = &iblobstorage.TempBLOBKeyType{
			ClusterAppID: istructs.ClusterAppID_sys_blobber,
			WSID:         bm.WSID(),
			SUUID:        bm.SUUID(),
		}
	default:
		// notest
		panic(fmt.Sprintf("unexpected blob operation: %d", bm.BLOBOperation()))
	}

	stateWriterDiscard := func(state iblobstorage.BLOBState) error {
		if state.Status != iblobstorage.BLOBStatus_Completed {
			return errors.New("blob is not completed")
		}
		if len(state.Error) > 0 {
			return errors.New(state.Error)
		}
		bm.Resp().Header().Set(coreutils.ContentType, state.Descr.MimeType)
		bm.Resp().Header().Add("Content-Disposition", fmt.Sprintf(`attachment;filename="%s"`, state.Descr.Name))
		bm.Resp().WriteHeader(http.StatusOK)
		return nil
	}
	if err := blobStorage.ReadBLOB(bm.RequestCtx(), blobKey, stateWriterDiscard, bm.Resp(), iblobstoragestg.RLimiter_Null); err != nil {
		logger.Error(fmt.Sprintf("failed to read or send BLOB: id %s, appQName %s, wsid %d: %s", blobKey.ID(), bm.AppQName(), bm.WSID(), err.Error()))
		if errors.Is(err, iblobstorage.ErrBLOBNotFound) {
			coreutils.ReplyErrf(bm.Sender(), http.StatusNotFound, err.Error())
			return
		}
		coreutils.ReplyErr(bm.Sender(), err)
	}
}
