/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * Aleksei Ponomarev
 */

package in10nmemv1

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/in10n"
	istructs "github.com/voedger/voedger/pkg/istructs"
)

type callbackMock struct {
	wait chan UpdateUnit
}

func TestBasicUsage(t *testing.T) {
	var wg sync.WaitGroup
	c := new(callbackMock)
	c.wait = make(chan UpdateUnit)

	projectionKeyExample := in10n.ProjectionKey{
		App:        istructs.AppQName_test1_app1,
		Projection: appdef.NewQName("test", "restaurant"),
		WS:         istructs.WSID(0),
	}

	quotasExample := in10n.Quotas{
		Channels:                1,
		ChannelsPerSubject:      1,
		Subscriptions:           1,
		SubscriptionsPerSubject: 1,
	}
	req := require.New(t)
	ctx, cancel := context.WithCancel(context.Background())

	broker := Provide(quotasExample)

	var channel in10n.ChannelID
	t.Run("Create channel.", func(t *testing.T) {
		var subject istructs.SubjectLogin = "paa"
		var err error
		channel, err = broker.NewChannel(subject, 24*time.Hour)
		req.NoError(err)
		req.NotNil(channel)
	})

	t.Run("Check channel count. count must be 1.", func(t *testing.T) {
		numChannels := broker.MetricNumChannels()
		req.Equal(1, numChannels)
	})

	wg.Add(1)
	go func() {
		defer wg.Done()
		broker.WatchChannel(ctx, channel, c.updatesMock)
	}()

	t.Run("Subscribe on projection.", func(t *testing.T) {
		var notExistsChannel = "NotExistChannel"
		// Try to subscribe on projection in not exist channel
		// must receive error ErrChannelNotExists
		err := broker.Subscribe(in10n.ChannelID(notExistsChannel), projectionKeyExample)
		req.ErrorIs(err, in10n.ErrChannelDoesNotExist)

		// check subscriptions, numSubscriptions must be equal 0
		numSubscriptions := broker.MetricNumSubcriptions()
		req.Equal(0, numSubscriptions)

		// Subscribe on exist channel numSubscriptions must be equal 1
		require.NoError(t, broker.Subscribe(channel, projectionKeyExample))
		numSubscriptions = broker.MetricNumSubcriptions()
		req.Equal(1, numSubscriptions)

		// Unsubscribe from not exist channel, raise error in10n.ErrChannelDoesNotExist
		err = broker.Unsubscribe("Not exists channel", projectionKeyExample)
		req.ErrorIs(err, in10n.ErrChannelDoesNotExist)

		// Unsubscribe from exist channel
		err = broker.Unsubscribe(channel, projectionKeyExample)
		req.NoError(err)
		// After unsubscribe numSubscriptions must be equal 0
		numSubscriptions = broker.MetricNumSubcriptions()
		req.Equal(0, numSubscriptions)

		// Subscribe on exist channel numSubscriptions must be equal 1
		require.NoError(t, broker.Subscribe(channel, projectionKeyExample))
		numSubscriptions = broker.MetricNumSubcriptions()
		req.Equal(1, numSubscriptions)

	})

	broker.Update(projectionKeyExample, istructs.Offset(122))
	broker.Update(projectionKeyExample, istructs.Offset(123))
	broker.Update(projectionKeyExample, istructs.Offset(124))
	broker.Update(projectionKeyExample, istructs.Offset(125))
	broker.Update(projectionKeyExample, istructs.Offset(126))

	for update := range c.wait {
		if update.Offset == istructs.Offset(126) {
			break
		}
	}
	cancel()
	wg.Wait()
}

func (c *callbackMock) updatesMock(projection in10n.ProjectionKey, offset istructs.Offset) {
	var unit = UpdateUnit{
		Projection: projection,
		Offset:     offset,
	}
	c.wait <- unit
}

// Try watch on not exists channel. WatchChannel must exit.
func TestWatchNotExistsChannel(t *testing.T) {
	req := require.New(t)

	quotasExample := in10n.Quotas{
		Channels:                1,
		ChannelsPerSubject:      1,
		Subscriptions:           1,
		SubscriptionsPerSubject: 1,
	}

	broker := Provide(quotasExample)
	ctx := context.TODO()

	t.Run("Create channel.", func(t *testing.T) {
		var subject istructs.SubjectLogin = "paa"
		channel, err := broker.NewChannel(subject, 24*time.Hour)
		req.NoError(err)
		req.NotNil(channel)
	})

	t.Run("Try watch not exist channel", func(t *testing.T) {
		req.Panics(func() {
			broker.WatchChannel(ctx, "not exist channel id", nil)
		}, "When try watch not exists channel - must panics")

	})
}

func TestQuotas(t *testing.T) {
	req := require.New(t)
	quotasExample := in10n.Quotas{
		Channels:                100,
		ChannelsPerSubject:      10,
		Subscriptions:           1000,
		SubscriptionsPerSubject: 100,
	}

	t.Run("Test channel quotas per subject. We create more channels than allowed for subject.", func(t *testing.T) {
		broker := Provide(quotasExample)
		for i := 0; i <= 10; i++ {
			_, err := broker.NewChannel("paa", 24*time.Hour)
			if i == 10 {
				req.ErrorIs(err, in10n.ErrQuotaExceeded_ChannelsPerSubject)
			}
		}
	})

	t.Run("Test channel quotas for the whole service. We create more channels than allowed for service.", func(t *testing.T) {
		broker := Provide(quotasExample)
		var subject istructs.SubjectLogin
		for i := 0; i < 10; i++ {
			subject = istructs.SubjectLogin("paa" + strconv.Itoa(i))
			for c := 0; c <= 10; c++ {
				_, err := broker.NewChannel(subject, 24*time.Hour)
				if i == 9 && c == 10 {
					req.ErrorIs(err, in10n.ErrQuotaExceeded_Channels)
				}
			}
		}
	})

	t.Run("Test subscription quotas for the whole service. We create more subscription than allowed for service.", func(t *testing.T) {
		projectionKeyExample := in10n.ProjectionKey{
			App:        istructs.AppQName_test1_app1,
			Projection: appdef.NewQName("test", "restaurant"),
			WS:         istructs.WSID(1),
		}
		broker := Provide(quotasExample)
		var subject istructs.SubjectLogin
		for i := 0; i < 100; i++ {
			subject = istructs.SubjectLogin("paa" + strconv.Itoa(i))
			channel, err := broker.NewChannel(subject, 24*time.Hour)
			req.NoError(err)
			for g := 0; g < 10; g++ {
				projectionKeyExample.WS = istructs.WSID(i + g)
				err = broker.Subscribe(channel, projectionKeyExample)
				req.NoError(err)
				if i == 99 && g == 9 {
					numSubscriptions := broker.MetricNumSubcriptions()
					req.Equal(1000, numSubscriptions)
					projectionKeyExample.WS = istructs.WSID(i + 100000)
					err = broker.Subscribe(channel, projectionKeyExample)
					req.ErrorIs(err, in10n.ErrQuotaExceeded_Subsciptions)
				}
			}
		}

	})

}
