/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package in10n

import (
	"context"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	istructs "github.com/voedger/voedger/pkg/istructs"
)

// Design: N10n https://dev.heeus.io/launchpad/#!13813
// This interface is provided only once per process
// Provide() must have a parameter of Quotas type
// Workflow (ref. https://golang.org/ref/spec for syntax):
//
//	???/n10n/channel: NewChannel() {Subscribe()} WatchChannel()
//	???/n10n/subscribe: Subscribe()
//	???/n10n/unsubscribe: Unsubscribe()
type IN10nBroker interface {

	// Errors: ErrQuotaExceeded_Channels*
	// @ConcurrentAccess
	NewChannel(subject istructs.SubjectLogin, channelDuration time.Duration) (channelID ChannelID, err error)

	// ChannelID must be taken from NewChannel()
	// Errors: ErrChannelDoesNotExist, ErrQuotaExceeded_Subsciptions*
	// @ConcurrentAccess
	Subscribe(channelID ChannelID, projection ProjectionKey) (err error)

	// Channel with ChannelID must exist (panic)
	// If channelDuration expired WatchChannel terminates
	// When WatchChannel enters/exits Metrics must be updated
	// It is not guaranteed that all offsets from Update() comes to `notifySubscriber` callback, some can be missed
	// If ctx is Done function must exit
	// @ConcurrentAccess
	WatchChannel(ctx context.Context, channelID ChannelID, notifySubscriber func(projection ProjectionKey, offset istructs.Offset))

	// This method MUST NOT BLOCK longer than 500 ns
	// Updates all channels which subscribed for this projection
	// @ConcurrentAccess
	Update(projection ProjectionKey, offset istructs.Offset)

	// ChannelID must be taken from NewChannel()
	// Errors: ErrChannelDoesNotExist
	// @ConcurrentAccess
	Unsubscribe(channelID ChannelID, projection ProjectionKey) (err error)

	// Metrics

	// @ConcurrentAccess
	MetricNumChannels() int
	// @ConcurrentAccess
	MetricNumSubcriptions() int
	// @ConcurrentAccess
	MetricSubject(ctx context.Context, cb func(subject istructs.SubjectLogin, numChannels int, numSubscriptions int))
}

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
