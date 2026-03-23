/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package n10n

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/istructs"
)

func TestSubscribeExtraLogging(t *testing.T) {
	logCap := logger.StartCapture(t, logger.LogLevelVerbose)

	require := require.New(t)

	projKey := in10n.ProjectionKey{
		App:        appdef.NewAppQName("test", "app"),
		Projection: appdef.NewQName("test", "View"),
		WS:         istructs.WSID(42),
	}
	wp := newN10nWP("chan-sub-1", projKey)

	require.NoError(logSubscribeSuccess(context.Background(), wp))

	logCap.HasLine("stage=n10n.subscribe.success",
		"vapp=test/app", "wsid=42",
		"projection=test.View", "channelid=chan-sub-1")
}
