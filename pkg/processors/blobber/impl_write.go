/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package blobprocessor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/iblobstorage"
	"github.com/voedger/voedger/pkg/istructs"
	ibus "github.com/voedger/voedger/staging/src/github.com/untillpro/airs-ibus"
)

func writeBLOB(bm IBLOBMessage, blobStorage iblobstorage.IBLOBStorage,
	busTimeout time.Duration, bus ibus.IBus, name string, mimeType string,
	body io.Reader) (blobID istructs.RecordID) {
	// request VVM for check the principalToken and get a blobID
	ok := false
	if ok, blobID = registerBLOB(bm.RequestCtx(), bm.WSID(), bm.AppQName().String(), "c.sys.UploadBLOBHelper",
		bm.Header(), busTimeout, bus, bm.Sender()); !ok {
		return
	}

	// write the BLOB
	key := iblobstorage.PersistentBLOBKeyType{
		ClusterAppID: istructs.ClusterAppID_sys_blobber,
		WSID:         bm.WSID(),
		BlobID:       blobID,
	}

	descr := iblobstorage.DescrType{
		Name:     name,
		MimeType: mimeType,
	}

	wLimiter := bm.WLimiterFactory()()
	if err := blobStorage.WriteBLOB(bm.RequestCtx(), key, descr, body, wLimiter); err != nil {
		if errors.Is(err, iblobstorage.ErrBLOBSizeQuotaExceeded) {
			coreutils.ReplyErrf(bm.Sender(), http.StatusForbidden, err.Error())
			return 0
		}
		coreutils.ReplyErr(bm.Sender(), err)
		return 0
	}

	// set WDoc<sys.BLOB>.status = BLOBStatus_Completed
	req := ibus.Request{
		Method:   ibus.HTTPMethodPOST,
		WSID:     bm.WSID(),
		AppQName: bm.AppQName().String(),
		Resource: "c.sys.CUD",
		Body:     []byte(fmt.Sprintf(`{"cuds":[{"sys.ID": %d,"fields":{"status":%d}}]}`, blobID, iblobstorage.BLOBStatus_Completed)),
		Header:   bm.Header(),
		Host:     coreutils.Localhost,
	}
	cudWDocBLOBUpdateResp, _, _, err := bus.SendRequest2(bm.RequestCtx(), req, busTimeout)
	if err != nil {
		coreutils.ReplyInternalServerError(bm.Sender(), "failed to exec c.sys.CUD", err)
		return 0
	}
	if cudWDocBLOBUpdateResp.StatusCode != http.StatusOK {
		coreutils.ReplyErrf(bm.Sender(), cudWDocBLOBUpdateResp.StatusCode, "c.sys.CUD returned error: "+string(cudWDocBLOBUpdateResp.Data))
		return 0
	}

	return blobID
}

func registerBLOB(ctx context.Context, wsid istructs.WSID, appQName string, registerFuncName string, header map[string][]string, busTimeout time.Duration,
	bus ibus.IBus, sender ibus.ISender) (ok bool, blobID istructs.RecordID) {
	req := ibus.Request{
		Method:   ibus.HTTPMethodPOST,
		WSID:     wsid,
		AppQName: appQName,
		Resource: registerFuncName,
		Body:     []byte(`{}`),
		Header:   header,
		Host:     coreutils.Localhost,
	}
	blobHelperResp, _, _, err := bus.SendRequest2(ctx, req, busTimeout)
	if err != nil {
		coreutils.ReplyInternalServerError(sender, "failed to exec "+registerFuncName, err)
		return false, istructs.NullRecordID
	}
	if blobHelperResp.StatusCode != http.StatusOK {
		coreutils.ReplyErrf(sender, blobHelperResp.StatusCode, fmt.Sprintf("%s returned error: %s", registerFuncName, string(blobHelperResp.Data)))
		return false, istructs.NullRecordID
	}

	cmdResp := map[string]interface{}{}
	if err := json.Unmarshal(blobHelperResp.Data, &cmdResp); err != nil {
		coreutils.ReplyInternalServerError(sender, "failed to json-unmarshal "+registerFuncName, err)
		return false, istructs.NullRecordID
	}
	newIDsIntf, ok := cmdResp["NewIDs"]
	if ok {
		newIDs := newIDsIntf.(map[string]interface{})
		return true, istructs.RecordID(newIDs["1"].(float64))
	}
	return true, istructs.NullRecordID
}
