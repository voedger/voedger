/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package blobprocessor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/goutils/testingu"
	"github.com/voedger/voedger/pkg/iblobstorage"
	"github.com/voedger/voedger/pkg/istructs"
)

type syncBuf struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (sb *syncBuf) Write(p []byte) (int, error) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.Write(p)
}

func (sb *syncBuf) String() string {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.String()
}

func (sb *syncBuf) Reset() {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	sb.buf.Reset()
}

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

func okCmdSender() bus.IRequestSender {
	return bus.NewIRequestSender(testingu.MockTime, func(_ context.Context, _ bus.Request, responder bus.IResponder) {
		_ = responder.Respond(bus.ResponseMeta{StatusCode: http.StatusOK}, `{"NewIDs":{"1":42}}`)
	})
}

func TestBlobWritePipeline(t *testing.T) {
	defer logger.SetLogLevelWithRestore(logger.LogLevelVerbose)()
	var buf syncBuf
	logger.SetCtxWriters(&buf, &buf)
	defer logger.SetCtxWriters(os.Stdout, os.Stderr)

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

		out := buf.String()
		require.Contains(out, "bp.meta")
		require.Contains(out, "testfile.txt")
		require.Contains(out, fmt.Sprintf(`ownerqname="%s"`, notApplicableInAPIv1))
		require.Contains(out, fmt.Sprintf(`ownerfield="%s"`, notApplicableInAPIv1))
		require.Contains(out, "bp.register.success")
		require.Contains(out, "blobid=")
		require.Contains(out, "bp.write.success")
		require.Contains(out, "bp.setcompleted.success")
	})

	t.Run("error", func(t *testing.T) {
		require := require.New(t)
		buf.Reset()
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

		out := buf.String()
		require.Contains(out, "bp.register.success")
		require.Contains(out, "bp.error")
		require.Contains(out, "storage write failure")
		require.Equal(http.StatusInternalServerError, capturedErr.HTTPStatus)
	})
}
