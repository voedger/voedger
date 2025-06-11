/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * Aleksei Ponomarev
 *
 * Copyright (c) 2023-present unTill Pro, Ltd.
 * @author Maxim Geraskin
 * Deep refactoring, no timers
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package in10nmem

import (
	"context"
	"log"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/testingu"
	"github.com/voedger/voedger/pkg/goutils/timeu"
	"github.com/voedger/voedger/pkg/in10n"
	istructs "github.com/voedger/voedger/pkg/istructs"
)

type callbackMock struct {
	data chan UpdateUnit
}

func (c *callbackMock) updatesMock(projection in10n.ProjectionKey, offset istructs.Offset) {
	var unit = UpdateUnit{
		Projection: projection,
		Offset:     offset,
	}
	c.data <- unit
}

func Test_SubscribeUnsubscribe(t *testing.T) {
	var wg sync.WaitGroup

	cb1 := new(callbackMock)
	cb1.data = make(chan UpdateUnit, 1)

	cb2 := new(callbackMock)
	cb2.data = make(chan UpdateUnit, 1)

	ctx, cancel := context.WithCancel(context.Background())

	projectionKey1 := in10n.ProjectionKey{
		App:        istructs.AppQName_test1_app1,
		Projection: appdef.NewQName("test", "restaurant"),
		WS:         istructs.WSID(8),
	}
	projectionKey2 := in10n.ProjectionKey{
		App:        istructs.AppQName_test1_app1,
		Projection: appdef.NewQName("test", "restaurant2"),
		WS:         istructs.WSID(9),
	}

	quotasExample := in10n.Quotas{
		Channels:                10,
		ChannelsPerSubject:      10,
		Subscriptions:           10,
		SubscriptionsPerSubject: 10,
	}
	req := require.New(t)

	nb, cleanup := ProvideEx2(quotasExample, timeu.NewITime())
	defer cleanup()

	var channel1ID in10n.ChannelID
	t.Run("Create and subscribe channel 1", func(t *testing.T) {
		var subject istructs.SubjectLogin = "paa"
		var err error
		channel1ID, err = nb.NewChannel(subject, 24*time.Hour)
		req.NoError(err)

		err = nb.Subscribe(channel1ID, projectionKey1)
		req.NoError(err)

		err = nb.Subscribe(channel1ID, projectionKey2)
		req.NoError(err)

		wg.Add(1)
		go func() {
			nb.WatchChannel(ctx, channel1ID, cb1.updatesMock)
			wg.Done()
		}()
	})

	var channel2ID in10n.ChannelID
	t.Run("Create and subscribe channel 2", func(t *testing.T) {
		var subject istructs.SubjectLogin = "paa"
		var err error
		channel2ID, err = nb.NewChannel(subject, 24*time.Hour)
		req.NoError(err)

		err = nb.Subscribe(channel2ID, projectionKey1)
		req.NoError(err)

		err = nb.Subscribe(channel2ID, projectionKey2)
		req.NoError(err)

		wg.Add(1)
		go func() {
			nb.WatchChannel(ctx, channel2ID, cb2.updatesMock)
			wg.Done()
		}()
	})

	// Update and see data

	for i := 1; i < 10; i++ {
		nb.Update(projectionKey1, istructs.Offset(i))
		<-cb1.data
		<-cb2.data
		nb.Update(projectionKey2, istructs.Offset(i))
		<-cb1.data
		<-cb2.data
	}

	// Unsubscribe all channels from projectionKey1

	nb.Unsubscribe(channel1ID, projectionKey1)
	nb.Unsubscribe(channel2ID, projectionKey1)

	for i := 100; i < 110; i++ {

		nb.Update(projectionKey2, istructs.Offset(i))
		<-cb1.data
		<-cb2.data

		nb.Update(projectionKey1, istructs.Offset(i))
		select {
		case <-cb1.data:
			t.Error("cb1.data must be empty")
		default:
			// TODO note that cb1.data may come later, should wait for broker idleness somehow
		}
		select {
		case <-cb2.data:
			t.Error("cb2.data must be empty")
			// TODO See note above
		default:
		}
	}
	cancel()
	wg.Wait()

}

// Test that after subscribing to a channel, the client receives updates for all projections with the current offset
// even if the projections were not explicitly updated after the subscription.
func Test_Subscribe_NoUpdate_Unsubscribe(t *testing.T) {

	var wg sync.WaitGroup

	cb1 := new(callbackMock)
	cb1.data = make(chan UpdateUnit, 1)

	ctx, cancel := context.WithCancel(context.Background())

	projectionKey1 := in10n.ProjectionKey{
		App:        istructs.AppQName_test1_app1,
		Projection: appdef.NewQName("test", "restaurant"),
		WS:         istructs.WSID(8),
	}
	projectionKey2 := in10n.ProjectionKey{
		App:        istructs.AppQName_test1_app1,
		Projection: appdef.NewQName("test", "restaurant2"),
		WS:         istructs.WSID(9),
	}

	quotasExample := in10n.Quotas{
		Channels:                10,
		ChannelsPerSubject:      10,
		Subscriptions:           10,
		SubscriptionsPerSubject: 10,
	}
	req := require.New(t)

	nb, cleanup := ProvideEx2(quotasExample, timeu.NewITime())
	defer cleanup()

	nb.Update(projectionKey1, istructs.Offset(100))
	nb.Update(projectionKey2, istructs.Offset(200))

	// Create channel
	var channel1ID in10n.ChannelID
	{
		var subject istructs.SubjectLogin = "paa"
		var err error
		channel1ID, err = nb.NewChannel(subject, 24*time.Hour)
		req.NoError(err)

		wg.Add(1)
		go func() {
			nb.WatchChannel(ctx, channel1ID, cb1.updatesMock)
			wg.Done()
		}()

	}

	for range 10 {
		// Subscribe
		{
			err := nb.Subscribe(channel1ID, projectionKey1)
			req.NoError(err)

			err = nb.Subscribe(channel1ID, projectionKey2)
			req.NoError(err)

		}
		// Should see some offsets

		{
			var ui []UpdateUnit
			ui = append(ui, <-cb1.data)
			ui = append(ui, <-cb1.data)
			req.Contains(ui, UpdateUnit{
				Offset:     istructs.Offset(100),
				Projection: projectionKey1,
			})
			req.Contains(ui, UpdateUnit{
				Offset:     istructs.Offset(200),
				Projection: projectionKey2,
			})
		}
		// Unsubscribe
		nb.Unsubscribe(channel1ID, projectionKey1)
		nb.Unsubscribe(channel1ID, projectionKey2)

	}

	cancel()
	wg.Wait()

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

	broker, cleanup := ProvideEx2(quotasExample, timeu.NewITime())
	defer cleanup()
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
		broker, cleanup := ProvideEx2(quotasExample, timeu.NewITime())
		defer cleanup()
		for i := 0; i <= 10; i++ {
			_, err := broker.NewChannel("paa", 24*time.Hour)
			if i == 10 {
				req.ErrorIs(err, in10n.ErrQuotaExceeded_ChannelsPerSubject)
			}
		}
	})

	t.Run("Test channel quotas for the whole service. We create more channels than allowed for service.", func(t *testing.T) {
		broker, cleanup := ProvideEx2(quotasExample, timeu.NewITime())
		defer cleanup()
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
		broker, cleanup := ProvideEx2(quotasExample, timeu.NewITime())
		defer cleanup()
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
					req.ErrorIs(err, in10n.ErrQuotaExceeded_Subscriptions)
				}
			}
		}

	})

}

// Flow:
// - Create mockTimer
// - Subscribe to QNameHeartbeat30
// - Start goroutine that will call WatchChannel(..notifySubscriber..)
// - mockTimer.FireNextTimerImmediately()
// - Make sure that notifySubscriber is called
func TestHeartbeats(t *testing.T) {

	req := require.New(t)
	mockTime := testingu.MockTime
	mockTime.FireNextTimerImmediately()

	quotasExample := in10n.Quotas{
		Channels:                1,
		ChannelsPerSubject:      1,
		Subscriptions:           1,
		SubscriptionsPerSubject: 1,
	}

	broker, cleanup := ProvideEx2(quotasExample, mockTime)
	defer cleanup()

	// Create channel and subscribe to Heartbeat30
	subject := istructs.SubjectLogin("testuser")
	channelID, err := broker.NewChannel(subject, 24*time.Hour)
	req.NoError(err)

	err = broker.Subscribe(channelID, in10n.Heartbeat30ProjectionKey)
	req.NoError(err)

	// Setup callback to receive updates
	cb := new(callbackMock)
	cb.data = make(chan UpdateUnit, 1)

	// Create context with cancel for cleanup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start watching the channel in a separate goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		broker.WatchChannel(ctx, channelID, cb.updatesMock)
	}()

	for range 10 {
		// Wait for update with timeout
		select {
		case update := <-cb.data:
			// Verify we got an update for the heartbeat projection
			req.Equal(in10n.Heartbeat30ProjectionKey, update.Projection)
			log.Println("Received heartbeat update:", update)
			mockTime.Add(30 * time.Second) // Simulate passage of time
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for heartbeat notification")
		}
	}

	// Clean up
	cancel()
	wg.Wait()
}

func TestChannelExpiration(t *testing.T) {
	quotasExample := in10n.Quotas{
		Channels:                1,
		ChannelsPerSubject:      1,
		Subscriptions:           1,
		SubscriptionsPerSubject: 1,
	}

	broker, cleanup := ProvideEx2(quotasExample, testingu.MockTime)
	defer cleanup()

	subject := istructs.SubjectLogin("test")
	channelID, err := broker.NewChannel(subject, time.Second)
	require.NoError(t, err)
	projectionKeyExample := in10n.ProjectionKey{
		App:        istructs.AppQName_test1_app1,
		Projection: appdef.NewQName("test", "restaurant"),
		WS:         istructs.WSID(1),
	}
	err = broker.Subscribe(channelID, projectionKeyExample)
	require.NoError(t, err)
	ctx := context.Background()
	wg := sync.WaitGroup{}
	wg.Add(1)
	eventHandled := make(chan any)
	go func() {
		broker.WatchChannel(ctx, channelID, func(projection in10n.ProjectionKey, offset istructs.Offset) {
			eventHandled <- nil
		})
		wg.Done()
	}()

	// check the notifications work
	broker.Update(projectionKeyExample, 42)
	<-eventHandled

	// expire the channel
	testingu.MockTime.Sleep(2 * time.Second)

	// try to send an event -> validation should fail because the channel is expired
	broker.Update(projectionKeyExample, 43)

	// expect WatchChannel() is done
	// observe "channel time to live expired: subjectlogin test" message in the log
	wg.Wait()
}

// Flow:
// - Create a channel
// - Start watching the channel
// - Subscribe to the projection1
// - Wait for metrics to be updated
// - Subscribe to the projection2
// - Wait for metrics to be updated
// - Unsubscribe from the projection1
// - Wait for metrics to be updated
// - Close context
// - Wait for metrics to be updated (zero)
func Test_MetricNumProjectionSubscriptions(t *testing.T) {
	req := require.New(t)

	quotasExample := in10n.Quotas{
		Channels:                1,
		ChannelsPerSubject:      1,
		Subscriptions:           10,
		SubscriptionsPerSubject: 10,
	}

	broker, cleanup := ProvideEx2(quotasExample, timeu.NewITime())
	defer cleanup()

	// Initially, no subscriptions should exist
	req.Equal(0, broker.MetricNumSubcriptions())

	// Create a channel
	subject := istructs.SubjectLogin("testuser")
	channelID, err := broker.NewChannel(subject, 24*time.Hour)
	req.NoError(err)

	// Setup projection keys
	projection1 := in10n.ProjectionKey{
		App:        istructs.AppQName_test1_app1,
		Projection: appdef.NewQName("test", "restaurant1"),
		WS:         istructs.WSID(1),
	}
	projection2 := in10n.ProjectionKey{
		App:        istructs.AppQName_test1_app1,
		Projection: appdef.NewQName("test", "restaurant2"),
		WS:         istructs.WSID(2),
	}

	// Setup callback and context for watching
	cb := new(callbackMock)
	cb.data = make(chan UpdateUnit, 10)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start watching the channel
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		broker.WatchChannel(ctx, channelID, cb.updatesMock)
	}()

	// Subscribe to projection1
	err = broker.Subscribe(channelID, projection1)
	req.NoError(err)

	// Wait for metrics to be updated
	req.Equal(1, broker.MetricNumSubcriptions())
	reqEventuallyEqual(t, 1, func() int {
		return broker.MetricNumProjectionSubscriptions(projection1)
	})

	// Subscribe to projection2
	err = broker.Subscribe(channelID, projection2)
	req.NoError(err)

	// Wait for metrics to be updated
	req.Equal(2, broker.MetricNumSubcriptions())
	req.Equal(1, broker.MetricNumProjectionSubscriptions(projection1))
	reqEventuallyEqual(t, 1, func() int {
		return broker.MetricNumProjectionSubscriptions(projection2)
	})

	// Unsubscribe from projection1
	broker.Unsubscribe(channelID, projection1)

	// Wait for metrics to be updated
	req.Equal(1, broker.MetricNumSubcriptions())
	reqEventuallyEqual(t, 0, func() int {
		return broker.MetricNumProjectionSubscriptions(projection1)
	})

	// Close context (this should clean up remaining subscriptions)
	cancel()

	// Wait for WatchChannel to finish
	wg.Wait()

	// Wait for metrics to be updated
	req.Equal(0, broker.MetricNumSubcriptions())
	reqEventuallyEqual(t, 0, func() int {
		return broker.MetricNumProjectionSubscriptions(projection1)
	})
	reqEventuallyEqual(t, 0, func() int {
		return broker.MetricNumProjectionSubscriptions(projection2)
	})
}

// Wait for 1 seconds
func reqEventuallyEqual(t *testing.T, expected int, fn func() int) {
	t.Helper()
	for range 10 {
		if fn() == expected {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Errorf("In one second, expected %d, got %d", expected, fn())
}
