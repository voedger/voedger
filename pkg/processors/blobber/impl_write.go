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
			err = blobStorage.WriteBLOB(bw.blobMessage.RequestCtx(), *key, bw.descr, bw.blobMessageWrite.Reader(), wLimiter)
		} else {
			key := (bw.blobKey).(*iblobstorage.TempBLOBKeyType)
			err = blobStorage.WriteTempBLOB(ctx, *key, bw.descr, bw.blobMessageWrite.Reader(), wLimiter, bw.duration)
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
		if !bw.isPersistent() {
			// do not account statuses for temp blobs
			return nil
		}
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
			return fmt.Errorf("failed to exec %s: %w", bw.registerFuncName, err)
		}
		if blobHelperResp.StatusCode != http.StatusOK {
			return coreutils.NewHTTPErrorf(blobHelperResp.StatusCode, fmt.Sprintf("%s returned error: %s", bw.registerFuncName, string(blobHelperResp.Data)))
		}

		if bw.isPersistent() {
			cmdResp := map[string]interface{}{}
			if err := json.Unmarshal(blobHelperResp.Data, &cmdResp); err != nil {
				return fmt.Errorf("failed to json-unmarshal %s :%w", bw.registerFuncName, err)
			}
			newIDsIntf, ok := cmdResp["NewIDs"]
			if !ok {
				// notest
				return coreutils.NewHTTPErrorf(http.StatusInternalServerError, "missing NewIDs in "+bw.registerFuncName+" cmd response")
			}
			newIDs := newIDsIntf.(map[string]interface{})
			bw.newBLOBID = istructs.RecordID(newIDs["1"].(float64))
			return nil
		}
		bw.newSUUID = iblobstorage.NewSUUID()
		return nil
	}
}
