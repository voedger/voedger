/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package n10n

import (
	"context"
	"errors"
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/istructs"
)

func TestN10NErrorLogging(t *testing.T) {
	logCap := logger.StartCapture(t, logger.LogLevelVerbose)

	t.Run("n10n.error", func(_ *testing.T) {
		logCap.Reset()

		projKey := in10n.ProjectionKey{
			App:        appdef.NewAppQName("test", "app"),
			Projection: appdef.NewQName("test", "View"),
			WS:         istructs.WSID(42),
		}
		wp := newN10nWP("chan-1", projKey)

		reportError(wp, errors.New("test n10n failure"))

		logCap.HasLine("stage=n10n.error", "test n10n failure",
			"channelid=chan-1", "projection=test.View",
			"vapp=test/app", "wsid=42")
	})
}

type noopResponder struct{}

func (noopResponder) StreamJSON(int) bus.IResponseWriter  { return nil }
func (noopResponder) StreamEvents() bus.IResponseWriter   { return nil }
func (noopResponder) Respond(bus.ResponseMeta, any) error { return nil }

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
