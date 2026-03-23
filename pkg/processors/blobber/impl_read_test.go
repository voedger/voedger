/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package blobprocessor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/goutils/testingu"
	"github.com/voedger/voedger/pkg/iblobstorage"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/processors"
)

func TestBLOBReadLogging(t *testing.T) {
	logCap := logger.StartCapture(t, logger.LogLevelVerbose)

	t.Run("success", func(t *testing.T) {
		require := require.New(t)
		storage := &mockBlobStorage{
			queryState: iblobstorage.BLOBState{
				Status: iblobstorage.BLOBStatus_Completed,
				Descr:  iblobstorage.DescrType{Name: "test.txt", ContentType: "text/plain"},
				Size:   100,
			},
		}
		p := providePipeline(context.Background(), storage, nil)
		defer p.Close()

		msg := &implIBLOBMessage_Read{
			implIBLOBMessage_base: implIBLOBMessage_base{
				appQName:         istructs.AppQName_test1_app1,
				wsid:             1,
				requestCtx:       context.Background(),
				header:           map[string]string{},
				okResponseIniter: func(_ ...string) io.Writer { return io.Discard },
				errorResponder:   func(_ coreutils.SysError) {},
				done:             make(chan interface{}),
			},
			existingBLOBIDOrSUUID: "42",
		}
		bw := &blobWorkpiece{blobMessage: msg}

		require.NoError(p.SendSync(bw))

		logCap.HasLine("bp.success", "blobid=42",
			fmt.Sprintf(`ownerqname="%s"`, notApplicableInAPIv1),
			fmt.Sprintf(`ownerfield="%s"`, notApplicableInAPIv1),
			fmt.Sprintf(`ownerid="%s"`, notApplicableInAPIv1))
	})

	t.Run("apiv2_success", func(t *testing.T) {
		require := require.New(t)
		logCap.Reset()

		ownerRecord := appdef.NewQName("test", "Owner")
		ownerRecordField := "testField"
		ownerID := istructs.RecordID(100)

		sender := bus.NewIRequestSender(testingu.MockTime, func(_ context.Context, req bus.Request, responder bus.IResponder) {
			switch processors.APIPath(req.APIPath) {
			case processors.APIPath_Docs:
				rw := responder.StreamJSON(http.StatusOK)
				_ = rw.Write(map[string]interface{}{ownerRecordField: istructs.RecordID(42)})
				rw.Close(nil)
			case processors.APIPath_Queries:
				rw := responder.StreamJSON(http.StatusOK)
				rw.Close(nil)
			}
		})

		storage := &mockBlobStorage{
			queryState: iblobstorage.BLOBState{
				Status: iblobstorage.BLOBStatus_Completed,
				Descr:  iblobstorage.DescrType{Name: "test.txt", ContentType: "text/plain"},
				Size:   100,
			},
		}
		p := providePipeline(context.Background(), storage, nil)
		defer p.Close()

		msg := &implIBLOBMessage_Read{
			implIBLOBMessage_base: implIBLOBMessage_base{
				appQName:         istructs.AppQName_test1_app1,
				wsid:             1,
				requestCtx:       context.Background(),
				header:           map[string]string{},
				okResponseIniter: func(_ ...string) io.Writer { return io.Discard },
				errorResponder:   func(_ coreutils.SysError) {},
				done:             make(chan interface{}),
				requestSender:    sender,
				isAPIv2:          true,
			},
			ownerRecord:      ownerRecord,
			ownerRecordField: ownerRecordField,
			ownerID:          ownerID,
		}
		bw := &blobWorkpiece{blobMessage: msg}

		require.NoError(p.SendSync(bw))

		logCap.HasLine("bp.success", "blobid=42",
			"ownerqname=test.Owner",
			"ownerfield=testField",
			"ownerid=100")
	})

	t.Run("error", func(t *testing.T) {
		require := require.New(t)
		logCap.Reset()
		var capturedErr coreutils.SysError
		storage := &mockBlobStorage{
			queryState: iblobstorage.BLOBState{
				Status: iblobstorage.BLOBStatus_Completed,
				Descr:  iblobstorage.DescrType{Name: "test.txt", ContentType: "text/plain"},
				Size:   100,
			},
			readErr: errors.New("storage read failure"),
		}
		p := providePipeline(context.Background(), storage, nil)
		defer p.Close()

		msg := &implIBLOBMessage_Read{
			implIBLOBMessage_base: implIBLOBMessage_base{
				appQName:         istructs.AppQName_test1_app1,
				wsid:             1,
				requestCtx:       context.Background(),
				header:           map[string]string{},
				okResponseIniter: func(_ ...string) io.Writer { return io.Discard },
				errorResponder:   func(se coreutils.SysError) { capturedErr = se },
				done:             make(chan interface{}),
			},
			existingBLOBIDOrSUUID: "42",
		}
		bw := &blobWorkpiece{blobMessage: msg}

		require.NoError(p.SendSync(bw))

		logCap.HasLine("bp.error", "storage read failure")
		require.Equal(http.StatusInternalServerError, capturedErr.HTTPStatus)
	})
}
