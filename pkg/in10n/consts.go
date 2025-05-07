/*
 * Copyright (c) 2025-present unTill Software Development Group B. V.
 * @author Maxim Geraskin
 */

package in10n

import (
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

// QNameHeartbeat30 is the name of the singleton that is used to simulate updates
var QNameHeartbeat30 = appdef.NewQName(appdef.SysPackage, "Heartbeat30")

const Heartbeat30Duration = 30 * time.Second

// [~server.n10n.heartbeats/freq.ZeroKey~impl]
var Heartbeat30ProjectionKey = ProjectionKey{
	App:        appdef.AppQName{},
	Projection: QNameHeartbeat30,
	WS:         istructs.NullWSID,
}
