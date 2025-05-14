/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 * @author Maxim Geraskin
 * Deep refactoring, no timers
 *
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * Aleksei Ponomarev
 * Initial implementation
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package in10nmem

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/goutils/timeu"
	"github.com/voedger/voedger/pkg/in10n"
	istructs "github.com/voedger/voedger/pkg/istructs"
)

type N10nBroker struct {
	sync.RWMutex
	projections      map[in10n.ProjectionKey]*projection
	channels         map[in10n.ChannelID]*channel
	quotas           in10n.Quotas
	metricBySubject  map[istructs.SubjectLogin]*metricType
	numSubscriptions int
	time             timeu.ITime
	events           chan event
}

type event struct {
	prj *projection
}

type projection struct {
	sync.Mutex

	offsetPointer *istructs.Offset

	toSubscribe map[in10n.ChannelID]*channel

	// merged by pnotifier using toSubscribe, toUnsubscribe
	subscribedChannels map[in10n.ChannelID]*channel
}

type subscription struct {
	deliveredOffset istructs.Offset
	currentOffset   *istructs.Offset
}

type channel struct {
	subject         istructs.SubjectLogin
	subscriptions   map[in10n.ProjectionKey]*subscription
	channelDuration time.Duration
	createTime      time.Time
	cchan           chan struct{}
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
	channel := channel{
		subject:         subject,
		subscriptions:   make(map[in10n.ProjectionKey]*subscription),
		channelDuration: channelDuration,
		createTime:      nb.time.Now(),
		cchan:           make(chan struct{}, 1),
	}
	nb.channels[channelID] = &channel
	return channelID, err
}

// Implementation of in10n.IN10nBroker
// Errors: ErrChannelDoesNotExist, ErrQuotaExceeded_Subscriptions*
//
// [~server.n10n.heartbeats/freq.ZeroKey~doc]:
// - If Subscribe is called for QNameHeartbeat30:
//   - ProjectionKey.WSID is set 0
//   - ProjectionKey.AppQName is set to {"", ""}
//
// [~server.n10n.heartbeats/freq.Interval30Seconds~doc]
// - Implementation generates a heartbeat every 30 seconds for all channels that are subscribed on QNameHeartbeat30
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
		return in10n.ErrQuotaExceeded_Subscriptions
	}
	if metric.numSubscriptions >= nb.quotas.SubscriptionsPerSubject {
		return in10n.ErrQuotaExceeded_SubscriptionsPerSubject
	}

	// [~server.n10n.heartbeats/freq.ZeroKey~impl]
	// [~server.n10n.heartbeats/freq.SingleNotification~impl]
	if projectionKey.Projection == in10n.QNameHeartbeat30 {
		projectionKey = in10n.Heartbeat30ProjectionKey
	}

	subscription := subscription{
		deliveredOffset: istructs.Offset(0),
		currentOffset:   guaranteeProjection(nb.projections, projectionKey),
	}
	channel.subscriptions[projectionKey] = &subscription
	metric.numSubscriptions++
	nb.numSubscriptions++

	{
		// Must exist because we create it in guaranteeProjection
		prj := nb.projections[projectionKey]
		prj.Lock()
		defer prj.Unlock()
		prj.toSubscribe[channelID] = channel
	}

	return err
}

func (nb *N10nBroker) Unsubscribe(channelID in10n.ChannelID, projectionKey in10n.ProjectionKey) (err error) {
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
	delete(channel.subscriptions, projectionKey)
	metric.numSubscriptions--
	nb.numSubscriptions--

	prj := nb.projections[projectionKey]
	if prj != nil {
		prj.Lock()
		defer prj.Unlock()
		prj.toSubscribe[channelID] = nil
	}

	return err
}

// Implementation of the in10n.IN10nBroker
func (nb *N10nBroker) WatchChannel(ctx context.Context, channelID in10n.ChannelID, notifySubscriber func(projection in10n.ProjectionKey, offset istructs.Offset)) {
	// check that the channelID with the given ChannelID exists
	channel, metric := func() (*channel, *metricType) {
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

	if logger.IsTrace() {
		logger.Trace("notified", channelID, channel.subject)
	}

	updateUnits := make([]UpdateUnit, 0)

	// cycle for channel.cchan and ctx
	for ctx.Err() == nil {
		select {
		case <-ctx.Done():
			break
		case <-channel.cchan:
			if logger.IsTrace() {
				logger.Trace("notified: ", channelID)
			}

			if ctx.Err() != nil {
				return
			}

			err := nb.validateChannel(channel)
			if err != nil {
				logger.Error(fmt.Sprintf("%s: subjectlogin %s", err.Error(), channel.subject))
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
				if logger.IsTrace() {
					logTrace("before notifySubscriber", unit.Projection, unit.Offset)
				}
				notifySubscriber(unit.Projection, unit.Offset)
			}
			updateUnits = updateUnits[:0]
		}

	}
}

func notifier(ctx context.Context, wg *sync.WaitGroup, events chan event) {
	defer func() {
		logger.Info("notifier goroutine stopped")
		wg.Done()
	}()

	logger.Info("notifier goroutine started")

	for ctx.Err() == nil {
		select {
		case <-ctx.Done():
			return
		case eve := <-events:
			prj := eve.prj

			// Actualize subscriptions
			{
				prj.Lock()
				for channelID, channel := range prj.toSubscribe {
					if channel != nil {
						prj.subscribedChannels[channelID] = channel
					} else {
						delete(prj.subscribedChannels, channelID)
					}
				}
				prj.Unlock()
			}

			// Notify subscribers
			if logger.IsTrace() {
				logger.Trace("notifier goroutine: len(prj.subscribedChannels):", strconv.Itoa(len(prj.subscribedChannels)))
			}
			for _, ch := range prj.subscribedChannels {
				select {
				case ch.cchan <- struct{}{}:
					if logger.IsTrace() {
						logger.Trace("notifier goroutine: ch.cchan <- struct{}{}")
					}
				default:
					if logger.IsVerbose() {
						logger.Verbose("notifier goroutine: channel full, skipping send")
					}
				}
			}
		}
	}

}

func guaranteeProjection(projections map[in10n.ProjectionKey]*projection, projectionKey in10n.ProjectionKey) (offsetPointer *istructs.Offset) {
	prj := projections[projectionKey]
	if prj == nil {
		prj = &projection{
			subscribedChannels: make(map[in10n.ChannelID]*channel),
			offsetPointer:      new(istructs.Offset),
			toSubscribe:        make(map[in10n.ChannelID]*channel),
		}
		projections[projectionKey] = prj

	}
	return prj.offsetPointer
}

// Update @ConcurrentAccess
// Update projections map with new offset
func (nb *N10nBroker) Update(projection in10n.ProjectionKey, offset istructs.Offset) {
	nb.Lock()
	*guaranteeProjection(nb.projections, projection) = offset
	prj := nb.projections[projection]
	nb.Unlock()

	e := event{prj: prj}
	nb.events <- e
	if logger.IsTrace() {
		logTrace("Update() completed", projection, offset)
	}
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

func NewN10nBroker(quotas in10n.Quotas, time timeu.ITime) (nb *N10nBroker, cleanup func()) {
	broker := N10nBroker{
		projections:     make(map[in10n.ProjectionKey]*projection),
		channels:        make(map[in10n.ChannelID]*channel),
		metricBySubject: make(map[istructs.SubjectLogin]*metricType),
		quotas:          quotas,
		time:            time,
		events:          make(chan event, eventsChannelSize),
	}
	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	cleanup = func() {
		cancel()
		wg.Wait()
	}

	wg.Add(1)
	go notifier(ctx, &wg, broker.events)
	wg.Add(1)
	go broker.heartbeat30(ctx, &wg)

	return &broker, cleanup
}

// Call Update() every 30 seconds for i10n.Heartbeat30ProjectionKey
func (nb *N10nBroker) heartbeat30(ctx context.Context, wg *sync.WaitGroup) {
	defer func() {
		logger.Info("heartbeat30 goroutine stopped")
		wg.Done()
	}()

	// [~server.n10n.heartbeats/freq.Interval30Seconds~impl]
	ticker := nb.time.NewTimerChan(in10n.Heartbeat30Duration)
	logger.Info("heartbeat30 goroutine started, Heartbeat30Duration:", in10n.Heartbeat30Duration)

	offset := istructs.Offset(1)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker:
			if logger.IsTrace() {
				logger.Trace("ticker")
			}
			nb.Update(in10n.Heartbeat30ProjectionKey, offset)
			offset++
			ticker = nb.time.NewTimerChan(in10n.Heartbeat30Duration)
		}
	}
}

func (nb *N10nBroker) validateChannel(channel *channel) error {
	nb.RLock()
	defer nb.RUnlock()
	// if channel lifetime > channelDuration defined in NewChannel when create channel - must exit
	if time.Since(channel.createTime) > channel.channelDuration {
		return ErrChannelExpired
	}
	return nil
}
