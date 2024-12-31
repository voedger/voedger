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
	"net/http"
	"time"

	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/iblobstorage"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/pipeline"
	ibus "github.com/voedger/voedger/staging/src/github.com/untillpro/airs-ibus"
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

func provideWriteBLOB(blobStorage iblobstorage.IBLOBStorage, wLimiterFactory WLimiterFactory) func(ctx context.Context, work pipeline.IWorkpiece) (err error) {
	return func(ctx context.Context, work pipeline.IWorkpiece) (err error) {
		bw := work.(*blobWorkpiece)
		wLimiter := wLimiterFactory()
		if bw.isPersistent() {
			key := (bw.blobKey).(*iblobstorage.PersistentBLOBKeyType)
			err = blobStorage.WriteBLOB(bw.blobMessage.RequestCtx(), *key, bw.descr, bw.blobMessage.Reader(), wLimiter)
		} else {
			key := (bw.blobKey).(*iblobstorage.TempBLOBKeyType)
			err = blobStorage.WriteTempBLOB(ctx, *key, bw.descr, bw.blobMessage.Reader(), wLimiter, bw.duration)
		}
		if errors.Is(err, iblobstorage.ErrBLOBSizeQuotaExceeded) {
			return coreutils.NewHTTPError(http.StatusForbidden, err)
		}
		return err
	}
}

func provideSetBLOBStatusCompleted(bus ibus.IBus, busTimeout time.Duration) func(ctx context.Context, work pipeline.IWorkpiece) (err error) {
	return func(ctx context.Context, work pipeline.IWorkpiece) (err error) {
		bw := work.(*blobWorkpiece)
		// set WDoc<sys.BLOB>.status = BLOBStatus_Completed
		req := ibus.Request{
			Method:   ibus.HTTPMethodPOST,
			WSID:     bw.blobMessage.WSID(),
			AppQName: bw.blobMessage.AppQName().String(),
			Resource: "c.sys.CUD",
			Body:     []byte(fmt.Sprintf(`{"cuds":[{"sys.ID": %d,"fields":{"status":%d}}]}`, bw.newBLOBID, iblobstorage.BLOBStatus_Completed)),
			Header:   bw.blobMessage.Header(),
			Host:     coreutils.Localhost,
		}
		cudWDocBLOBUpdateResp, _, _, err := bus.SendRequest2(bw.blobMessage.RequestCtx(), req, busTimeout)
		if err != nil {
			return fmt.Errorf("failed to exec c.sys.CUD: %w", err)
		}
		if cudWDocBLOBUpdateResp.StatusCode != http.StatusOK {
			return coreutils.NewHTTPErrorf(cudWDocBLOBUpdateResp.StatusCode, "c.sys.CUD returned error: ", string(cudWDocBLOBUpdateResp.Data))
		}
		return nil
	}
}

func provideRegisterBLOB(bus ibus.IBus, busTimeout time.Duration) func(ctx context.Context, work pipeline.IWorkpiece) (err error) {
	return func(ctx context.Context, work pipeline.IWorkpiece) (err error) {
		bw := work.(*blobWorkpiece)
		req := ibus.Request{
			Method:   ibus.HTTPMethodPOST,
			WSID:     bw.blobMessage.WSID(),
			AppQName: bw.blobMessage.AppQName().String(),
			Resource: bw.registerFuncName,
			Body:     []byte(`{}`),
			Header:   bw.blobMessage.Header(),
			Host:     coreutils.Localhost,
		}
		blobHelperResp, _, _, err := bus.SendRequest2(ctx, req, busTimeout)
		if err != nil {
			bw.blobMessage.ReplyError(http.StatusInternalServerError, fmt.Sprintf("failed to exec %s: %s", bw.registerFuncName, err))
		}
		if blobHelperResp.StatusCode != http.StatusOK {
			bw.blobMessage.ReplyError(blobHelperResp.StatusCode,
				fmt.Sprintf("%s returned error: %s", bw.registerFuncName, string(blobHelperResp.Data)))
		}

		cmdResp := map[string]interface{}{}
		if err := json.Unmarshal(blobHelperResp.Data, &cmdResp); err != nil {
			bw.blobMessage.ReplyError(http.StatusInternalServerError, fmt.Sprintf("failed to json-unmarshal %s :%s", bw.registerFuncName, err))
		}
		newIDsIntf, ok := cmdResp["NewIDs"]
		if ok {
			newIDs := newIDsIntf.(map[string]interface{})
			bw.newBLOBID = istructs.RecordID(newIDs["1"].(float64))
			return nil
		}
		// notest
		return coreutils.NewHTTPErrorf(http.StatusInternalServerError, "missing NewIDs in "+bw.registerFuncName+" cmd response")
	}
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
