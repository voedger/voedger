/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package n10n

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/istructs"
)

func TestSubscribeAndWatchLogging(t *testing.T) {
	logCap := logger.StartCapture(t, logger.LogLevelVerbose)

	t.Run("n10n.subscribe&watch.success single key", func(t *testing.T) {
		require := require.New(t)
		logCap.Reset()

		projKey := in10n.ProjectionKey{
			App:        appdef.NewAppQName("test", "app"),
			Projection: appdef.NewQName("test", "View"),
			WS:         istructs.WSID(42),
		}
		wp := newN10nWP("chan-1", projKey)

		require.NoError(logSubscribeAndWatchSuccess(context.Background(), wp))

		logCap.HasLine("stage=n10n.subscribe&watch.success",
			"vapp=test/app", "wsid=42",
			"projection=test.View", "channelid=chan-1")
	})

	t.Run("n10n.subscribe&watch.success multi key", func(t *testing.T) {
		require := require.New(t)
		logCap.Reset()

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

		logCap.HasLine(
			"stage=n10n.subscribe&watch.success",
			"channelid=chan-2",
			"projection=test.View1",
			"wsid=1",
		)
		logCap.HasLine("stage=n10n.subscribe&watch.success",
			"channelid=chan-2",
			"projection=test.View2",
			"wsid=2",
		)
	})

	t.Run("n10n.sse_send.success", func(t *testing.T) {
		logCap.Reset()

		projKey := in10n.ProjectionKey{
			App:        appdef.NewAppQName("test", "app"),
			Projection: appdef.NewQName("test", "View"),
			WS:         istructs.WSID(42),
		}
		wp := newN10nWP("chan-3", projKey)
		projCtx := n10nProjectionLogCtx(wp.logCtx, projKey)

		logger.VerboseCtx(projCtx, "n10n.sse_send.success", "event: test data: 100  ")

		logCap.HasLine("stage=n10n.sse_send.success",
			"channelid=chan-3", "vapp=test/app",
			"wsid=42", "projection=test.View")
	})

	t.Run("n10n.sse_send.error", func(t *testing.T) {
		logCap.Reset()

		projKey := in10n.ProjectionKey{
			App:        appdef.NewAppQName("test", "app"),
			Projection: appdef.NewQName("test", "View"),
			WS:         istructs.WSID(42),
		}
		wp := newN10nWP("chan-4", projKey)
		projCtx := n10nProjectionLogCtx(wp.logCtx, projKey)

		logger.ErrorCtx(projCtx, "n10n.sse_send.error", errors.New("write failed"))

		logCap.HasLine("stage=n10n.sse_send.error", "write failed",
			"channelid=chan-4", "vapp=test/app",
			"wsid=42", "projection=test.View")
	})

	t.Run("n10n.watch.done", func(t *testing.T) {
		require := require.New(t)
		logCap.Reset()

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

		logCap.HasLine("stage=n10n.watch.done",
			"channelid=chan-5", "vapp=test/app",
			"wsid=42", "projection=test.View")
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
