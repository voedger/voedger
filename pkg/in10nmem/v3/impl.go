/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package in10nmemv3

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/istructs"
)

type N10nBroker struct {
	sync.RWMutex
	projections map[in10n.ProjectionKey]*projectionType
	channels    map[in10n.ChannelID]*channelType
}

type projectionValue struct {
	offset  istructs.Offset
	sigchan chan struct{}
}

type projectionType struct {
	value atomic.Value
}

type channelType struct {
	pvalue *atomic.Value
	pkey   in10n.ProjectionKey
}

func (nb *N10nBroker) NewChannel(subject istructs.SubjectLogin, channelDuration time.Duration) (channelID in10n.ChannelID, err error) {
	nb.Lock()
	defer nb.Unlock()
	channelID = in10n.ChannelID(uuid.New().String())
	channel := channelType{}
	nb.channels[channelID] = &channel
	return channelID, err
}

func (nb *N10nBroker) guaranteeProjection(projectionKey in10n.ProjectionKey) *projectionType {
	nb.Lock()
	defer nb.Unlock()
	prj := nb.projections[projectionKey]
	if prj == nil {
		prj = &projectionType{}
		prj.value.Store(projectionValue{offset: 0, sigchan: make(chan struct{})})
		nb.projections[projectionKey] = prj
	}
	return prj
}

func (nb *N10nBroker) Subscribe(channelID in10n.ChannelID, projectionKey in10n.ProjectionKey) (err error) {
	channel, channelOK := nb.channels[channelID]
	if !channelOK {
		return in10n.ErrChannelDoesNotExist
	}
	prj := nb.guaranteeProjection(projectionKey)
	channel.pvalue = &prj.value
	channel.pkey = projectionKey
	return
}

func (nb *N10nBroker) Update(projection in10n.ProjectionKey, offset istructs.Offset) {
	prj := nb.guaranteeProjection(projection)
	oldvalue := prj.value.Load().(projectionValue)
	prj.value.Store(projectionValue{offset: offset, sigchan: make(chan struct{})})
	close(oldvalue.sigchan)
}

func (nb *N10nBroker) MetricNumChannels() int {
	return len(nb.channels)
}

func (nb *N10nBroker) MetricNumSubcriptions() int {
	return 0
}

func (nb *N10nBroker) MetricSubject(ctx context.Context, cb func(subject istructs.SubjectLogin, numChannels int, numSubscriptions int)) {

}

func NewN10nBroker(quotas in10n.Quotas, now func() time.Time) (nb *N10nBroker, cleanup func()) {
	return &N10nBroker{
			projections: make(map[in10n.ProjectionKey]*projectionType),
			channels:    make(map[in10n.ChannelID]*channelType),
		},
		func() {}
}

func (nb *N10nBroker) Unsubscribe(channelID in10n.ChannelID, projectionKey in10n.ProjectionKey) (err error) {
	return
}

func (nb *N10nBroker) WatchChannel(ctx context.Context, channelID in10n.ChannelID, notifySubscriber func(projection in10n.ProjectionKey, offset istructs.Offset)) {
	nb.Lock()
	channel, channelOK := nb.channels[channelID]
	nb.Unlock()
	if !channelOK {
		panic(fmt.Errorf("channel with channelID: %s must exists %w", channelID, in10n.ErrChannelDoesNotExist))
	}

	reportedOffset := istructs.Offset(0)

forctx:
	for ctx.Err() == nil {
		v := channel.pvalue.Load().(projectionValue)
		if v.offset > reportedOffset {
			notifySubscriber(channel.pkey, v.offset)
			reportedOffset = v.offset
		}
		select {
		case <-ctx.Done():
			break forctx
		case <-v.sigchan:
		}

	}

}
