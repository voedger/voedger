/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package n10n

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/istructs"
)

func TestSubscribeAndWatchLogging(t *testing.T) {
	defer logger.SetLogLevelWithRestore(logger.LogLevelVerbose)()
	var buf syncBuf
	logger.SetCtxWriters(&buf, &buf)
	defer logger.SetCtxWriters(os.Stdout, os.Stderr)

	t.Run("n10n.subscribe&watch.success single key", func(t *testing.T) {
		require := require.New(t)
		buf.Reset()

		projKey := in10n.ProjectionKey{
			App:        appdef.NewAppQName("test", "app"),
			Projection: appdef.NewQName("test", "View"),
			WS:         istructs.WSID(42),
		}
		wp := newN10nWP("chan-1", projKey)

		require.NoError(logSubscribeAndWatchSuccess(context.Background(), wp))

		out := buf.String()
		require.Contains(out, "stage=n10n.subscribe&watch.success")
		require.Contains(out, "vapp=test/app")
		require.Contains(out, "wsid=42")
		require.Contains(out, "projection=test.View")
		require.Contains(out, "channelid=chan-1")
	})

	t.Run("n10n.subscribe&watch.success multi key", func(t *testing.T) {
		require := require.New(t)
		buf.Reset()

		projKey1 := in10n.ProjectionKey{
			App:        appdef.NewAppQName("test", "app"),
			Projection: appdef.NewQName("test", "View1"),
			WS:         istructs.WSID(1),
		}
		projKey2 := in10n.ProjectionKey{
			App:        appdef.NewAppQName("test", "app"),
			Projection: appdef.NewQName("test", "View2"),
			WS:         istructs.WSID(2),
		}
		wp := &n10nWorkpiece{
			channelID:  "chan-2",
			requestCtx: context.Background(),
			logCtx:     context.Background(),
			appQName:   appdef.NewAppQName("test", "app"),
			responder:  noopResponder{},
		}
		wp.logCtx = logger.WithContextAttrs(wp.logCtx, map[string]any{
			logAttr_ChannelID: "chan-2",
		})
		wp.subscribedProjectionKeys = []in10n.ProjectionKey{projKey1, projKey2}

		require.NoError(logSubscribeAndWatchSuccess(context.Background(), wp))

		out := buf.String()
		require.Contains(out, "channelid=chan-2")
		require.Contains(out, "projection=test.View1")
		require.Contains(out, "wsid=1")
		require.Contains(out, "projection=test.View2")
		require.Contains(out, "wsid=2")
	})

	t.Run("n10n.sse_send.success", func(t *testing.T) {
		require := require.New(t)
		buf.Reset()

		projKey := in10n.ProjectionKey{
			App:        appdef.NewAppQName("test", "app"),
			Projection: appdef.NewQName("test", "View"),
			WS:         istructs.WSID(42),
		}
		wp := newN10nWP("chan-3", projKey)
		projCtx := n10nProjectionLogCtx(wp.logCtx, projKey)

		logger.VerboseCtx(projCtx, "n10n.sse_send.success", "event: test data: 100  ")

		out := buf.String()
		require.Contains(out, "stage=n10n.sse_send.success")
		require.Contains(out, "channelid=chan-3")
		require.Contains(out, "vapp=test/app")
		require.Contains(out, "wsid=42")
		require.Contains(out, "projection=test.View")
	})

	t.Run("n10n.sse_send.error", func(t *testing.T) {
		require := require.New(t)
		buf.Reset()

		projKey := in10n.ProjectionKey{
			App:        appdef.NewAppQName("test", "app"),
			Projection: appdef.NewQName("test", "View"),
			WS:         istructs.WSID(42),
		}
		wp := newN10nWP("chan-4", projKey)
		projCtx := n10nProjectionLogCtx(wp.logCtx, projKey)

		logger.ErrorCtx(projCtx, "n10n.sse_send.error", errors.New("write failed"))

		out := buf.String()
		require.Contains(out, "stage=n10n.sse_send.error")
		require.Contains(out, "write failed")
		require.Contains(out, "channelid=chan-4")
		require.Contains(out, "vapp=test/app")
		require.Contains(out, "wsid=42")
		require.Contains(out, "projection=test.View")
	})

	t.Run("n10n.watch.done", func(t *testing.T) {
		require := require.New(t)
		buf.Reset()

		projKey := in10n.ProjectionKey{
			App:        appdef.NewAppQName("test", "app"),
			Projection: appdef.NewQName("test", "View"),
			WS:         istructs.WSID(42),
		}
		wp := newN10nWP("chan-5", projKey)
		wp.channelCleanup = func() {}
		wp.responseWriter = &noopResponseWriter{}

		p := &implIN10NProc{n10nBroker: &immediateWatchBroker{}}
		require.NoError(p.watchChannel(context.Background(), wp))
		p.goroutinesWG.Wait()

		out := buf.String()
		require.Contains(out, "stage=n10n.watch.done")
		require.Contains(out, "channelid=chan-5")
		require.Contains(out, "vapp=test/app")
		require.Contains(out, "wsid=42")
		require.Contains(out, "projection=test.View")
	})
}

type noopResponseWriter struct{}

func (noopResponseWriter) Write(_ any) error { return nil }
func (noopResponseWriter) Close(_ error)     {}

type immediateWatchBroker struct{}

func (immediateWatchBroker) NewChannel(_ istructs.SubjectLogin, _ time.Duration) (in10n.ChannelID, func(), error) {
	return "", func() {}, nil
}
func (immediateWatchBroker) Subscribe(_ in10n.ChannelID, _ in10n.ProjectionKey) error { return nil }
func (immediateWatchBroker) WatchChannel(_ context.Context, _ in10n.ChannelID, _ func(in10n.ProjectionKey, istructs.Offset)) {
}
func (immediateWatchBroker) Update(_ in10n.ProjectionKey, _ istructs.Offset) {}
func (immediateWatchBroker) Unsubscribe(_ in10n.ChannelID, _ in10n.ProjectionKey) error {
	return nil
}
func (immediateWatchBroker) MetricNumChannels() int      { return 0 }
func (immediateWatchBroker) MetricNumSubscriptions() int { return 0 }
func (immediateWatchBroker) MetricSubject(_ context.Context, _ func(istructs.SubjectLogin, int, int)) {
}
func (immediateWatchBroker) MetricNumProjectionSubscriptions(_ in10n.ProjectionKey) int { return 0 }
