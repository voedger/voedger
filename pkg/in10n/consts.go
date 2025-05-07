/*
 * Copyright (c) 2025-present unTill Software Development Group B. V.
 * @author Maxim Geraskin
 */

package in10n

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

// qNameHeartbeat30 is the name of the singleton that is used to simulate updates
var qNameHeartbeat30 = appdef.NewQName(appdef.SysPackage, "Heartbeat30")

const in10nNullWSID = istructs.NullWSID

var heartbeat30ProjectionKey = ProjectionKey{
	App:        appdef.AppQName{},
	Projection: qNameHeartbeat30,
	WS:         in10nNullWSID,
}
