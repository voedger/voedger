/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package in10n

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

type ChannelID string
type SubscriptionID string

type ProjectionKey struct {
	App        appdef.AppQName
	Projection appdef.QName
	WS         istructs.WSID
}

type Quotas struct {
	Channels                int
	ChannelsPerSubject      int
	Subscriptions           int
	SubscriptionsPerSubject int
}
