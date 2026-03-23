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
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/goutils/httpu"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/goutils/testingu"
	"github.com/voedger/voedger/pkg/iblobstorage"
	"github.com/voedger/voedger/pkg/istructs"
)

type mockBlobStorage struct {
	iblobstorage.IBLOBStorage
	queryState iblobstorage.BLOBState
	readErr    error
	writeErr   error
}

func (m *mockBlobStorage) QueryBLOBState(_ context.Context, _ iblobstorage.IBLOBKey) (iblobstorage.BLOBState, error) {
	return m.queryState, nil
}

func (m *mockBlobStorage) ReadBLOB(_ context.Context, _ iblobstorage.IBLOBKey, _ func(iblobstorage.BLOBState) error, _ io.Writer, _ iblobstorage.RLimiterType) error {
	return m.readErr
}

func (m *mockBlobStorage) WriteBLOB(_ context.Context, _ iblobstorage.PersistentBLOBKeyType, _ iblobstorage.DescrType, _ io.Reader, _ iblobstorage.WLimiterType) (uint64, error) {
	if m.writeErr != nil {
		return 0, m.writeErr
	}
	return 100, nil
}

func (m *mockBlobStorage) WriteTempBLOB(_ context.Context, _ iblobstorage.TempBLOBKeyType, _ iblobstorage.DescrType, _ io.Reader, _ iblobstorage.WLimiterType, _ iblobstorage.DurationType) (uint64, error) {
	if m.writeErr != nil {
		return 0, m.writeErr
	}
	return 100, nil
}

func okCmdSender() bus.IRequestSender {
	return bus.NewIRequestSender(testingu.MockTime, func(_ context.Context, _ bus.Request, responder bus.IResponder) {
		_ = responder.Respond(bus.ResponseMeta{StatusCode: http.StatusOK}, `{"NewIDs":{"1":42}}`)
	})
}

func TestBLOBWriteLogging(t *testing.T) {
	logCap := logger.StartCapture(t, logger.LogLevelVerbose)

	t.Run("success", func(t *testing.T) {
		require := require.New(t)
		sender := okCmdSender()
		storage := &mockBlobStorage{}
		p := providePipeline(context.Background(), storage, func() iblobstorage.WLimiterType { return nil })
		defer p.Close()

		msg := &implIBLOBMessage_Write{
			implIBLOBMessage_base: implIBLOBMessage_base{
				appQName:         istructs.AppQName_test1_app1,
				wsid:             1,
				requestCtx:       context.Background(),
				header:           map[string]string{},
				requestSender:    sender,
				okResponseIniter: func(_ ...string) io.Writer { return io.Discard },
				errorResponder:   func(_ coreutils.SysError) {},
				done:             make(chan interface{}),
			},
			urlQueryValues: url.Values{"name": {"testfile.txt"}, "mimeType": {"text/plain"}},
			reader:         io.NopCloser(strings.NewReader("test data")),
		}
		bw := &blobWorkpiece{blobMessage: msg}

		require.NoError(p.SendSync(bw))

		logCap.HasLine("bp.meta", "testfile.txt",
			fmt.Sprintf(`ownerqname="%s"`, notApplicableInAPIv1),
			fmt.Sprintf(`ownerfield="%s"`, notApplicableInAPIv1))
		logCap.HasLine("bp.register.success", "blobid=")
		logCap.HasLine("bp.write.success")
		logCap.HasLine("bp.setcompleted.success")
	})

	t.Run("apiv2_success", func(t *testing.T) {
		require := require.New(t)
		logCap.Reset()

		ownerRecord := appdef.NewQName("test", "Owner")
		ownerRecordField := "testField"

		sender := bus.NewIRequestSender(testingu.MockTime, func(_ context.Context, _ bus.Request, responder bus.IResponder) {
			_ = responder.Respond(bus.ResponseMeta{StatusCode: http.StatusOK}, `{"NewIDs":{"1":42}}`)
		})
		storage := &mockBlobStorage{}
		p := providePipeline(context.Background(), storage, func() iblobstorage.WLimiterType { return nil })
		defer p.Close()

		msg := &implIBLOBMessage_Write{
			implIBLOBMessage_base: implIBLOBMessage_base{
				appQName:   istructs.AppQName_test1_app1,
				wsid:       1,
				requestCtx: context.Background(),
				header: map[string]string{
					coreutils.BlobName: "testfile.txt",
					httpu.ContentType:  "text/plain",
					"Ttl":              "1d",
				},
				requestSender:    sender,
				okResponseIniter: func(_ ...string) io.Writer { return io.Discard },
				errorResponder:   func(_ coreutils.SysError) {},
				done:             make(chan interface{}),
				isAPIv2:          true,
			},
			ownerRecord:      ownerRecord,
			ownerRecordField: ownerRecordField,
			reader:           io.NopCloser(strings.NewReader("test data")),
		}
		bw := &blobWorkpiece{blobMessage: msg}

		require.NoError(p.SendSync(bw))

		logCap.HasLine("bp.meta", "testfile.txt",
			"ownerqname=test.Owner",
			"ownerfield=testField")
		logCap.HasLine("bp.register.success", "blobid=",
			"ownerqname=test.Owner",
			"ownerfield=testField")
		logCap.HasLine("bp.write.success")
	})

	t.Run("error", func(t *testing.T) {
		require := require.New(t)
		logCap.Reset()
		var capturedErr coreutils.SysError
		sender := okCmdSender()
		storage := &mockBlobStorage{writeErr: errors.New("storage write failure")}
		p := providePipeline(context.Background(), storage, func() iblobstorage.WLimiterType { return nil })
		defer p.Close()

		msg := &implIBLOBMessage_Write{
			implIBLOBMessage_base: implIBLOBMessage_base{
				appQName:         istructs.AppQName_test1_app1,
				wsid:             1,
				requestCtx:       context.Background(),
				header:           map[string]string{},
				requestSender:    sender,
				okResponseIniter: func(_ ...string) io.Writer { return io.Discard },
				errorResponder:   func(se coreutils.SysError) { capturedErr = se },
				done:             make(chan interface{}),
			},
			urlQueryValues: url.Values{"name": {"testfile.txt"}, "mimeType": {"text/plain"}},
			reader:         io.NopCloser(strings.NewReader("test data")),
		}
		bw := &blobWorkpiece{blobMessage: msg}

		require.NoError(p.SendSync(bw))

		logCap.HasLine("bp.register.success")
		logCap.HasLine("bp.error", "storage write failure")
		require.Equal(http.StatusInternalServerError, capturedErr.HTTPStatus)
	})
}
