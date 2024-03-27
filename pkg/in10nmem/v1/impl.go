/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * Aleksei Ponomarev
 */

package in10nmemv1

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/voedger/voedger/pkg/in10n"
	istructs "github.com/voedger/voedger/pkg/istructs"
)

type N10nBroker struct {
	projections      map[in10n.ProjectionKey]*istructs.Offset
	channels         map[in10n.ChannelID]*channelType
	quotas           in10n.Quotas
	metricBySubject  map[istructs.SubjectLogin]*metricType
	numSubscriptions int
	now              func() time.Time
	sync.RWMutex
}

type projectionOffsets struct {
	deliveredOffset istructs.Offset
	currentOffset   *istructs.Offset
}

type channelType struct {
	subject         istructs.SubjectLogin
	subscriptions   map[in10n.ProjectionKey]*projectionOffsets
	channelDuration time.Duration
	createTime      time.Time
}

type metricType struct {
	numChannels      int
	numSubscriptions int
}

// NewChannel @ConcurrentAccess
// Create new channel.
// On timeout channel will be closed. channelDuration determines time during with it will be open.
func (nb *N10nBroker) NewChannel(subject istructs.SubjectLogin, channelDuration time.Duration) (channelID in10n.ChannelID, err error) {
	nb.Lock()
	defer nb.Unlock()
	var metric *metricType
	if len(nb.channels) >= nb.quotas.Channels {
		return "", in10n.ErrQuotaExceeded_Channels
	}
	metric = nb.metricBySubject[subject]
	if metric != nil {
		if metric.numChannels >= nb.quotas.ChannelsPerSubject {
			return "", in10n.ErrQuotaExceeded_ChannelsPerSubject
		}
	} else {
		metric = new(metricType)
		nb.metricBySubject[subject] = metric
	}
	metric.numChannels++
	channelID = in10n.ChannelID(uuid.New().String())
	channel := channelType{
		subject:         subject,
		subscriptions:   make(map[in10n.ProjectionKey]*projectionOffsets),
		channelDuration: channelDuration,
		createTime:      nb.now(),
	}
	nb.channels[channelID] = &channel
	return channelID, err
}

// Subscribe @ConcurrentAccess
// Subscribe to the channel for the projection. If channel does not exist: will return error ErrChannelNotExists
func (nb *N10nBroker) Subscribe(channelID in10n.ChannelID, projectionKey in10n.ProjectionKey) (err error) {
	nb.Lock()
	defer nb.Unlock()
	channel, channelOK := nb.channels[channelID]
	if !channelOK {
		return in10n.ErrChannelDoesNotExist
	}

	metric, metricOK := nb.metricBySubject[channel.subject]
	if !metricOK {
		return ErrMetricDoesNotExists
	}

	if nb.numSubscriptions >= nb.quotas.Subscriptions {
		return in10n.ErrQuotaExceeded_Subsciptions
	}
	if metric.numSubscriptions >= nb.quotas.SubscriptionsPerSubject {
		return in10n.ErrQuotaExceeded_SubsciptionsPerSubject
	}

	subscription := projectionOffsets{
		deliveredOffset: istructs.Offset(0),
		currentOffset:   guaranteeOffsetPointer(nb.projections, projectionKey),
	}
	channel.subscriptions[projectionKey] = &subscription
	metric.numSubscriptions++
	nb.numSubscriptions++
	return err
}

func (nb *N10nBroker) Unsubscribe(channelID in10n.ChannelID, projection in10n.ProjectionKey) (err error) {
	nb.Lock()
	defer nb.Unlock()

	channel, cOK := nb.channels[channelID]
	if !cOK {
		return in10n.ErrChannelDoesNotExist
	}
	metric, mOK := nb.metricBySubject[channel.subject]
	if !mOK {
		return ErrMetricDoesNotExists
	}
	delete(channel.subscriptions, projection)
	metric.numSubscriptions--
	nb.numSubscriptions--
	return err
}

// WatchChannel @ConcurrentAccess
// Create WatchChannel for notify clients about changed projections. If channel for this demand does not exist or
// channel already watched - exit.
func (nb *N10nBroker) WatchChannel(ctx context.Context, channelID in10n.ChannelID, notifySubscriber func(projection in10n.ProjectionKey, offset istructs.Offset)) {
	// check that the channelID with the given ChannelID exists
	channel, metric := func() (*channelType, *metricType) {
		nb.RLock()
		defer nb.RUnlock()
		channel, channelOK := nb.channels[channelID]
		if !channelOK {
			panic(fmt.Errorf("channel with channelID: %s must exists %w", channelID, in10n.ErrChannelDoesNotExist))
		}
		metric, metricOK := nb.metricBySubject[channel.subject]
		if !metricOK {
			panic(fmt.Errorf("metric for channel with channelID: %s must exists", channelID))
		}
		return channel, metric
	}()

	defer func() {
		nb.Lock()
		metric.numChannels--
		metric.numSubscriptions -= len(channel.subscriptions)
		nb.numSubscriptions -= len(channel.subscriptions)
		delete(nb.channels, channelID)
		nb.Unlock()
	}()

	ticker := time.NewTicker(pollingInterval)
	defer ticker.Stop()

	updateUnits := make([]UpdateUnit, 0)
	for range ticker.C {

		if ctx.Err() != nil {
			return
		}

		err := nb.validateChannel(channel)
		if err != nil {
			return
		}

		// find projection for update and collect
		nb.Lock()
		for projection, channelOffsets := range channel.subscriptions {
			if *channelOffsets.currentOffset > channelOffsets.deliveredOffset {
				updateUnits = append(updateUnits,
					UpdateUnit{
						Projection: projection,
						Offset:     *channelOffsets.currentOffset,
					})
				channelOffsets.deliveredOffset = *channelOffsets.currentOffset
			}
		}
		nb.Unlock()
		for _, unit := range updateUnits {
			notifySubscriber(unit.Projection, unit.Offset)
		}
		updateUnits = updateUnits[:0]
	}
}

func guaranteeOffsetPointer(projections map[in10n.ProjectionKey]*istructs.Offset, projection in10n.ProjectionKey) (offsetPointer *istructs.Offset) {
	offsetPointer = projections[projection]
	if offsetPointer == nil {
		offsetPointer = new(istructs.Offset)
		projections[projection] = offsetPointer
	}
	return
}

// Update @ConcurrentAccess
// Update projections map with new offset
func (nb *N10nBroker) Update(projection in10n.ProjectionKey, offset istructs.Offset) {
	nb.Lock()
	defer nb.Unlock()
	*guaranteeOffsetPointer(nb.projections, projection) = offset
}

// MetricNumChannels @ConcurrentAccess
// return channels count
func (nb *N10nBroker) MetricNumChannels() int {
	nb.RLock()
	defer nb.RUnlock()
	return len(nb.channels)
}

func (nb *N10nBroker) MetricNumSubcriptions() int {
	nb.RLock()
	defer nb.RUnlock()
	return nb.numSubscriptions
}

func (nb *N10nBroker) MetricSubject(ctx context.Context, cb func(subject istructs.SubjectLogin, numChannels int, numSubscriptions int)) {
	postMetric := func(subject istructs.SubjectLogin, metric *metricType) (err error) {
		nb.RLock()
		defer nb.RUnlock()
		cb(subject, metric.numChannels, metric.numSubscriptions)
		return err
	}
	for subject, subjectMetric := range nb.metricBySubject {
		if ctx.Err() != nil {
			return
		}
		err := postMetric(subject, subjectMetric)
		if err != nil {
			return
		}
	}
}

func NewN10nBroker(quotas in10n.Quotas, now func() time.Time) *N10nBroker {
	broker := N10nBroker{
		projections:     make(map[in10n.ProjectionKey]*istructs.Offset),
		channels:        make(map[in10n.ChannelID]*channelType),
		metricBySubject: make(map[istructs.SubjectLogin]*metricType),
		quotas:          quotas,
		now:             now,
	}
	return &broker
}

func (nb *N10nBroker) validateChannel(channel *channelType) error {
	nb.RLock()
	defer nb.RUnlock()
	// if channel lifetime > channelDuration defined in NewChannel when create channel - must exit
	if nb.Since(channel.createTime) > channel.channelDuration {
		return ErrChannelExpired
	}
	return nil
}

func (nb *N10nBroker) Since(t time.Time) time.Duration {
	return nb.now().Sub(t)
}
