/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package n10n

import (
	"bytes"
	"context"
	"errors"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/istructs"
)

func TestN10NErrorLogging(t *testing.T) {
	defer logger.SetLogLevelWithRestore(logger.LogLevelVerbose)()
	var buf syncBuf
	logger.SetCtxWriters(&buf, &buf)
	defer logger.SetCtxWriters(os.Stdout, os.Stderr)

	t.Run("n10n.error", func(t *testing.T) {
		require := require.New(t)
		buf.Reset()

		projKey := in10n.ProjectionKey{
			App:        appdef.NewAppQName("test", "app"),
			Projection: appdef.NewQName("test", "View"),
			WS:         istructs.WSID(42),
		}
		wp := newN10nWP("chan-1", projKey)

		reportError(wp, errors.New("test n10n failure"))

		out := buf.String()
		require.Contains(out, "stage=n10n.error")
		require.Contains(out, "test n10n failure")
		require.Contains(out, "channelid=chan-1")
		require.Contains(out, "projection=test.View")
		require.Contains(out, "vapp=test/app")
		require.Contains(out, "wsid=42")
	})
}

type noopResponder struct{}

func (noopResponder) StreamJSON(int) bus.IResponseWriter  { return nil }
func (noopResponder) StreamEvents() bus.IResponseWriter   { return nil }
func (noopResponder) Respond(bus.ResponseMeta, any) error { return nil }

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

func newN10nWP(channelID string, projKey in10n.ProjectionKey) *n10nWorkpiece {
	wp := &n10nWorkpiece{
		channelID:                in10n.ChannelID(channelID),
		requestCtx:               context.Background(),
		logCtx:                   context.Background(),
		appQName:                 projKey.App,
		responder:                noopResponder{},
		subscribedProjectionKeys: []in10n.ProjectionKey{projKey},
	}
	baseCtx := logger.WithContextAttrs(wp.logCtx, map[string]any{
		logAttr_ChannelID: channelID,
	})
	wp.logCtx = n10nProjectionLogCtx(baseCtx, projKey)
	return wp
}
