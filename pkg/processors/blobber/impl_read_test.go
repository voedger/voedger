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
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/iblobstorage"
	"github.com/voedger/voedger/pkg/istructs"
)

func TestBlobReadPipeline(t *testing.T) {
	defer logger.SetLogLevelWithRestore(logger.LogLevelVerbose)()
	var buf syncBuf
	logger.SetCtxWriters(&buf, &buf)
	defer logger.SetCtxWriters(os.Stdout, os.Stderr)

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

		out := buf.String()
		require.Contains(out, "bp.success")
		require.Contains(out, "blobid=42")
		require.Contains(out, fmt.Sprintf(`ownerqname="%s"`, notApplicableInAPIv1))
		require.Contains(out, fmt.Sprintf(`ownerfield="%s"`, notApplicableInAPIv1))
		require.Contains(out, fmt.Sprintf(`ownerid="%s"`, notApplicableInAPIv1))
	})

	t.Run("error", func(t *testing.T) {
		require := require.New(t)
		buf.Reset()
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

		out := buf.String()
		require.Contains(out, "bp.error")
		require.Contains(out, "storage read failure")
		require.Equal(http.StatusInternalServerError, capturedErr.HTTPStatus)
	})
}
