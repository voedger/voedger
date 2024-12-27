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

func blobReadMessageHandler(bm IBLOBMessage, blobStorage iblobstorage.IBLOBStorage, bus ibus.IBus, busTimeout time.Duration) {
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
		WriteTextResponse(bbm.resp, "failed to exec q.sys.DownloadBLOBAuthnz: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if blobHelperResp.StatusCode != http.StatusOK {
		WriteTextResponse(bbm.resp, "q.sys.DownloadBLOBAuthnz returned error: "+string(blobHelperResp.Data), blobHelperResp.StatusCode)
		return
	}

	// read the BLOB
	var blobKey iblobstorage.IBLOBKey
	switch typedKey := blobReadDetails.(type) {
	case blobReadDetails_Persistent:
		blobKey = &iblobstorage.PersistentBLOBKeyType{
			ClusterAppID: istructs.ClusterAppID_sys_blobber,
			WSID:         bbm.wsid,
			BlobID:       typedKey.blobID,
		}
	case blobReadDetails_Temporary:
		blobKey = &iblobstorage.TempBLOBKeyType{
			ClusterAppID: istructs.ClusterAppID_sys_blobber,
			WSID:         bbm.wsid,
			SUUID:        typedKey.suuid,
		}
	default:
		// notest
		panic(fmt.Sprintf("unexpected blobReadDetails: %T", blobReadDetails))
	}
	stateWriterDiscard := func(state iblobstorage.BLOBState) error {
		if state.Status != iblobstorage.BLOBStatus_Completed {
			return errors.New("blob is not completed")
		}
		if len(state.Error) > 0 {
			return errors.New(state.Error)
		}
		bbm.resp.Header().Set(coreutils.ContentType, state.Descr.MimeType)
		bbm.resp.Header().Add("Content-Disposition", fmt.Sprintf(`attachment;filename="%s"`, state.Descr.Name))
		bbm.resp.WriteHeader(http.StatusOK)
		return nil
	}
	if err := blobStorage.ReadBLOB(bbm.req.Context(), blobKey, stateWriterDiscard, bbm.resp, iblobstoragestg.RLimiter_Null); err != nil {
		logger.Error(fmt.Sprintf("failed to read or send BLOB: id %s, appQName %s, wsid %d: %s", blobKey.ID(), bbm.appQName, bbm.wsid, err.Error()))
		if errors.Is(err, iblobstorage.ErrBLOBNotFound) {
			WriteTextResponse(bbm.resp, err.Error(), http.StatusNotFound)
			return
		}
		WriteTextResponse(bbm.resp, err.Error(), http.StatusInternalServerError)
	}
}
