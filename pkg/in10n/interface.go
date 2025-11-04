/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package in10n

import (
	"context"
	"time"

	istructs "github.com/voedger/voedger/pkg/istructs"
)

// This interface is provided only once per process
// Provide() must have a parameter of Quotas type
type IN10nBroker interface {

	// Errors: ErrQuotaExceeded_Channels*
	// @ConcurrentAccess
	NewChannel(subject istructs.SubjectLogin, channelDuration time.Duration) (channelID ChannelID, channelCleanup func(), err error)

	// ChannelID must be taken from NewChannel()
	// Errors: ErrChannelDoesNotExist, ErrQuotaExceeded_Subscriptions*
	//
	// [~server.n10n.heartbeats/freq.ZeroKey~doc]:
	// - If Subscribe is called for QNameHeartbeat30:
	//   - ProjectionKey.WSID is set 0
	//   - ProjectionKey.AppQName is set to {"", ""}
	//
	// [~server.n10n.heartbeats/freq.Interval30Seconds~doc]
	// - Implementation generates a heartbeat every 30 seconds for all channels that are subscribed on QNameHeartbeat30
	Subscribe(channelID ChannelID, projection ProjectionKey) (err error)

	// Panics if a Channel with ChannelID does not exist
	// Terminates if channelDuration expired
	// Metrics are updated when WatchChannel enters/exits
	// It is guaranteed that the client is eventually notified about the latest offset used in Update() for the
	//   projection, including the calls to Update() happened prior to the Subscribe() call
	// Exits if ctx is Done
	// Only one client must call WatchChannel, concurrent use is not allowed
	//
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
	MetricNumSubscriptions() int
	// @ConcurrentAccess
	MetricSubject(ctx context.Context, cb func(subject istructs.SubjectLogin, numChannels int, numSubscriptions int))

	// @ConcurrentAccess
	MetricNumProjectionSubscriptions(projection ProjectionKey) int
}
