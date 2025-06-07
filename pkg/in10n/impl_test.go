/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package in10n

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

func TestProjectionToJSON(t *testing.T) {
	pk := ProjectionKey{
		App:        appdef.NewAppQName("owner", "app"),
		Projection: appdef.NewQName("ownertab", "table"),
		WS:         istructs.WSID(42),
	}

	require.JSONEq(t, `{"App":"owner/app", "Projection":"ownertab.table", "WS":42}`, pk.ToJSON())
}
