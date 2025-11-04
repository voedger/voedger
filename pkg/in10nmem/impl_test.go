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
	"fmt"
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

var (
	quotasExample = in10n.Quotas{
		Channels:                10,
		ChannelsPerSubject:      10,
		Subscriptions:           10,
		SubscriptionsPerSubject: 10,
	}
	projectionKey1 = in10n.ProjectionKey{
		App:        istructs.AppQName_test1_app1,
		Projection: appdef.NewQName("test", "restaurant"),
		WS:         istructs.WSID(8),
	}
	projectionKey2 = in10n.ProjectionKey{
		App:        istructs.AppQName_test1_app1,
		Projection: appdef.NewQName("test", "restaurant2"),
		WS:         istructs.WSID(9),
	}
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

	vvmOrRequestCtx, vvmOrRequestCtxCancel := context.WithCancel(context.Background())

	req := require.New(t)

	nb, n10nCleanup := NewN10nBroker(quotasExample, timeu.NewITime())

	var channel1ID in10n.ChannelID
	var channel1Cleanup func()
	t.Run("Create and subscribe channel 1", func(t *testing.T) {
		var subject istructs.SubjectLogin = "paa"
		var err error
		channel1ID, channel1Cleanup, err = nb.NewChannel(subject, 24*time.Hour)
		req.NoError(err)

		err = nb.Subscribe(channel1ID, projectionKey1)
		req.NoError(err)

		err = nb.Subscribe(channel1ID, projectionKey2)
		req.NoError(err)

		wg.Add(1)
		go func() {
			nb.WatchChannel(vvmOrRequestCtx, channel1ID, cb1.updatesMock)
			wg.Done()
		}()
	})

	var channel2ID in10n.ChannelID
	var channel2Cleanup func()
	t.Run("Create and subscribe channel 2", func(t *testing.T) {
		var subject istructs.SubjectLogin = "paa"
		var err error
		channel2ID, channel2Cleanup, err = nb.NewChannel(subject, 24*time.Hour)
		req.NoError(err)

		err = nb.Subscribe(channel2ID, projectionKey1)
		req.NoError(err)

		err = nb.Subscribe(channel2ID, projectionKey2)
		req.NoError(err)

		wg.Add(1)
		go func() {
			nb.WatchChannel(vvmOrRequestCtx, channel2ID, cb2.updatesMock)
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

	require.NoError(t, nb.Unsubscribe(channel1ID, projectionKey1))
	require.NoError(t, nb.Unsubscribe(channel2ID, projectionKey1))

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
	vvmOrRequestCtxCancel()
	wg.Wait()

	if channel1Cleanup != nil {
		channel1Cleanup()
	}
	if channel2Cleanup != nil {
		channel2Cleanup()
	}

	checkMetricsZero(t, nb, projectionKey1, projectionKey2)
	n10nCleanup()
}

// Test that after subscribing to a channel, the client receives updates for all projections with the current offset
// even if the projections were not explicitly updated after the subscription.
func Test_Subscribe_NoUpdate_Unsubscribe(t *testing.T) {

	var wg sync.WaitGroup

	cb := new(callbackMock)
	cb.data = make(chan UpdateUnit, 1)

	ctx, cancel := context.WithCancel(context.Background())

	req := require.New(t)

	nb, n10nCleanup := NewN10nBroker(quotasExample, timeu.NewITime())

	nb.Update(projectionKey1, istructs.Offset(100))
	nb.Update(projectionKey2, istructs.Offset(200))

	// Create channel
	var channelID in10n.ChannelID
	var channelCleanup func()
	{
		var subject istructs.SubjectLogin = "paa"
		var err error
		channelID, channelCleanup, err = nb.NewChannel(subject, 24*time.Hour)
		req.NoError(err)

		wg.Add(1)
		go func() {
			nb.WatchChannel(ctx, channelID, cb.updatesMock)
			wg.Done()
		}()

	}

	// for range 10 {
	// Subscribe
	{
		err := nb.Subscribe(channelID, projectionKey1)
		req.NoError(err)

		err = nb.Subscribe(channelID, projectionKey2)
		req.NoError(err)

	}
	// Should see some offsets

	{
		var ui []UpdateUnit
		ui = append(ui, <-cb.data)
		ui = append(ui, <-cb.data)
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
	require.NoError(t, nb.Unsubscribe(channelID, projectionKey1))
	require.NoError(t, nb.Unsubscribe(channelID, projectionKey2))

	// }

	cancel()
	wg.Wait()

	channelCleanup()
	checkMetricsZero(t, nb, projectionKey1, projectionKey2)
	n10nCleanup()
}

// Try watch on not exists channel. WatchChannel must exit.
func TestWatchNotExistsChannel(t *testing.T) {
	req := require.New(t)

	broker, n10nCleanup := NewN10nBroker(quotasExample, timeu.NewITime())
	ctx := context.TODO()

	t.Run("Try watch not exist channel", func(t *testing.T) {
		req.Panics(func() {
			broker.WatchChannel(ctx, "not exist channel id", nil)
		}, "When try watch not exists channel - must panics")
	})
	checkMetricsZero(t, broker)
	n10nCleanup()
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
		broker, brokerCleanup := NewN10nBroker(quotasExample, timeu.NewITime())
		chanCleanups := []func(){}
		for i := 0; i <= 10; i++ {
			_, chanCleanup, err := broker.NewChannel("paa", 24*time.Hour)
			if i == 10 {
				req.ErrorIs(err, in10n.ErrQuotaExceeded_ChannelsPerSubject)
			} else {
				chanCleanups = append(chanCleanups, chanCleanup)
			}
		}
		for _, chanCleanup := range chanCleanups {
			chanCleanup()
		}
		checkMetricsZero(t, broker)
		brokerCleanup()
	})

	t.Run("Test channel quotas for the whole service. We create more channels than allowed for service.", func(t *testing.T) {
		broker, brokerCleanup := NewN10nBroker(quotasExample, timeu.NewITime())
		var subject istructs.SubjectLogin
		channelCleanups := []func(){}
		for i := 0; i < 10; i++ {
			subject = istructs.SubjectLogin("paa" + strconv.Itoa(i))
			for c := 0; c < 10; c++ {
				_, channelCleanup, err := broker.NewChannel(subject, 24*time.Hour)
				req.NoError(err)
				channelCleanups = append(channelCleanups, channelCleanup)
			}
		}
		// Try to create one more channel than allowed (quota is 100)
		_, _, err := broker.NewChannel("extraSubject", 24*time.Hour)
		req.ErrorIs(err, in10n.ErrQuotaExceeded_Channels)

		for _, channelCleanup := range channelCleanups {
			channelCleanup()
		}
		checkMetricsZero(t, broker)
		brokerCleanup()
	})

	t.Run("Test subscription quotas for the whole service. We create more subscription than allowed for service.", func(t *testing.T) {
		projectionKeyExample := in10n.ProjectionKey{
			App:        istructs.AppQName_test1_app1,
			Projection: appdef.NewQName("test", "restaurant"),
			WS:         istructs.WSID(1),
		}
		projections := []in10n.ProjectionKey{}
		broker, brokerCleanup := NewN10nBroker(quotasExample, timeu.NewITime())
		var subject istructs.SubjectLogin
		chanCleanups := []func(){}
		for i := 0; i < 100; i++ {
			subject = istructs.SubjectLogin("paa" + strconv.Itoa(i))
			channel, chanCleanup, err := broker.NewChannel(subject, 24*time.Hour)
			req.NoError(err)
			chanCleanups = append(chanCleanups, chanCleanup)
			for g := 0; g < 10; g++ {
				projectionKeyExample.WS = istructs.WSID(i + g)
				err = broker.Subscribe(channel, projectionKeyExample)
				req.NoError(err)
				projections = append(projections, projectionKeyExample)
				if i == 99 && g == 9 {
					numSubscriptions := broker.MetricNumSubscriptions()
					req.Equal(1000, numSubscriptions)
					projectionKeyExample.WS = istructs.WSID(i + 100000)
					err = broker.Subscribe(channel, projectionKeyExample)
					req.ErrorIs(err, in10n.ErrQuotaExceeded_Subscriptions)
				}
			}
		}
		for _, chanCleanup := range chanCleanups {
			chanCleanup()
		}
		checkMetricsZero(t, broker, projections...)
		brokerCleanup()
	})

}

// flow:
// - create mockTimer
// - subscribe to QNameHeartbeat30
// - start goroutine that will call WatchChannel(..notifySubscriber..)
// - test heartbeats interval
//   - advance the mock time by a second until Heartbeat30Duration-1
//   - expect no heartbeats on each second
//   - advance time by 1 second more
//   - expect heartbeat
//   - repeat 10 times
func TestHeartbeats(t *testing.T) {
	req := require.New(t)
	mockTime := testingu.NewMockTime()
	done := make(chan any)
	mockTime.SetOnNextTimerArmed(func() {
		close(done)
	})

	broker, brokerCleanup := NewN10nBroker(quotasExample, mockTime)

	// Create channel and subscribe to Heartbeat30
	subject := istructs.SubjectLogin("testuser")
	channelID, channelCleanup, err := broker.NewChannel(subject, 24*time.Hour)
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
		for range int(in10n.Heartbeat30Duration.Seconds()) - 1 {
			<-done
			mockTime.Add(time.Second)
			select {
			case <-cb.data:
				t.Fatal("Unexpected heartbeat")
			default:
				// OK, heartbeat time not come yet
			}
		}
		done = make(chan any)
		mockTime.SetOnNextTimerArmed(func() {
			close(done)
		})
		mockTime.Add(time.Second)
		select {
		case update := <-cb.data:
			req.Equal(in10n.Heartbeat30ProjectionKey, update.Projection)
			log.Println("Received heartbeat update:", update)
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for heartbeat notification")
		}
	}

	// Clean up
	cancel()
	wg.Wait()

	channelCleanup()
	checkMetricsZero(t, broker, in10n.Heartbeat30ProjectionKey)
	brokerCleanup()
}

func TestChannelExpiration(t *testing.T) {
	broker, brokerCleanup := NewN10nBroker(quotasExample, testingu.MockTime)

	subject := istructs.SubjectLogin("test")
	channelID, channelCleanup, err := broker.NewChannel(subject, time.Second)
	require.NoError(t, err)
	err = broker.Subscribe(channelID, projectionKey1)
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
	broker.Update(projectionKey1, 42)
	<-eventHandled

	// expire the channel
	testingu.MockTime.Sleep(2 * time.Second)

	// try to send an event -> validation should fail because the channel is expired
	broker.Update(projectionKey1, 43)

	// expect WatchChannel() is done
	// observe "channel time to live expired: subjectlogin test" message in the log
	wg.Wait()

	channelCleanup()
	checkMetricsZero(t, broker, projectionKey1)
	brokerCleanup()
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

	broker, brokerCleanup := NewN10nBroker(quotasExample, timeu.NewITime())

	// Initially, no subscriptions should exist
	req.Equal(0, broker.MetricNumSubscriptions())

	// Create a channel
	subject := istructs.SubjectLogin("testuser")
	channelID, channelCleanup, err := broker.NewChannel(subject, 24*time.Hour)
	req.NoError(err)

	// Setup projection keys

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
	err = broker.Subscribe(channelID, projectionKey1)
	req.NoError(err)

	// Wait for metrics to be updated
	req.Equal(1, broker.MetricNumSubscriptions())
	reqEventuallyNumProjectionSubscriptions(t, broker, 1, projectionKey1)

	// Subscribe to projection2
	err = broker.Subscribe(channelID, projectionKey2)
	req.NoError(err)

	// Wait for metrics to be updated
	req.Equal(2, broker.MetricNumSubscriptions())
	req.Equal(1, broker.MetricNumProjectionSubscriptions(projectionKey1))
	reqEventuallyNumProjectionSubscriptions(t, broker, 1, projectionKey2)

	// Unsubscribe from projection1
	err = broker.Unsubscribe(channelID, projectionKey1)
	require.NoError(t, err)

	// Wait for metrics to be updated
	req.Equal(1, broker.MetricNumSubscriptions())
	reqEventuallyNumProjectionSubscriptions(t, broker, 0, projectionKey1)

	// Close context (this should clean up remaining subscriptions)
	cancel()

	// Wait for WatchChannel to finish
	wg.Wait()

	channelCleanup()

	// Wait for metrics to be updated
	checkMetricsZero(t, broker, projectionKey1, projectionKey2)
	brokerCleanup()
}

// Wait for 1 seconds
func reqEventuallyNumProjectionSubscriptions(t *testing.T, broker in10n.IN10nBroker, expected int, proj in10n.ProjectionKey) {
	t.Helper()
	require.Eventually(t, func() bool {
		return broker.MetricNumProjectionSubscriptions(proj) == expected
	}, time.Second, 100*time.Millisecond)
}

func checkMetricsZero(t *testing.T, nb in10n.IN10nBroker, projections ...in10n.ProjectionKey) {
	t.Helper()
	req := require.New(t)
	req.Zero(nb.MetricNumSubscriptions())
	req.Zero(nb.MetricNumChannels())
	wg := sync.WaitGroup{}
	for _, prj := range projections {
		wg.Add(1)
		go func(prj in10n.ProjectionKey) {
			// TestQuotas creates ~1000 projections, so check it simultaneously
			reqEventuallyNumProjectionSubscriptions(t, nb, 0, prj)
			wg.Done()
		}(prj)
	}
	wg.Wait()
	nb.MetricSubject(context.Background(), func(subject istructs.SubjectLogin, numChannels, numSubscriptions int) {
		req.Zero(numChannels)
		req.Zero(numSubscriptions)
	})
}

func TestMultipleWatchChannelProtection(t *testing.T) {
	req := require.New(t)

	nb, brokerCleanup := NewN10nBroker(quotasExample, timeu.NewITime())

	channelID, channelCleanup, err := nb.NewChannel("testuser", 24*time.Hour)
	req.NoError(err)

	err = nb.Subscribe(channelID, projectionKey1)
	req.NoError(err)

	watchCtx, cancel := context.WithCancel(context.Background())
	panicChan := make(chan any, 2)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer func() {
			panicChan <- recover()
			wg.Done()
		}()
		nb.WatchChannel(watchCtx, channelID, func(projection in10n.ProjectionKey, offset istructs.Offset) {})
	}()
	wg.Add(1)
	go func() {
		defer func() {
			panicChan <- recover()
			wg.Done()
		}()
		nb.WatchChannel(watchCtx, channelID, func(projection in10n.ProjectionKey, offset istructs.Offset) {})
	}()

	panic1 := <-panicChan
	cancel()
	panic2 := <-panicChan
	wg.Wait()
	panicMessage := ""
	if panic1 != nil && panic2 != nil {
		t.Fatal("1 panic expected, got 2:", panic1, panic2)
	} else if panic1 != nil {
		panicMessage = fmt.Sprint(panic1)
	} else if panic2 != nil {
		panicMessage = fmt.Sprint(panic2)
	}
	require.Contains(t, panicMessage, in10n.ErrChannelAlreadyBeingWatched.Error())

	channelCleanup()
	brokerCleanup()
}

func TestDoubleSubscribeAndUnsubscribe(t *testing.T) {
	require := require.New(t)

	broker, brokerCleanup := NewN10nBroker(quotasExample, timeu.NewITime())

	testSubject := istructs.SubjectLogin("testuser")
	channelID, channelCleanup, err := broker.NewChannel(testSubject, 24*time.Hour)
	require.NoError(err)

	watchCtx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		broker.WatchChannel(watchCtx, channelID, func(projection in10n.ProjectionKey, offset istructs.Offset) {})
		wg.Done()
	}()

	// double subscribe
	err = broker.Subscribe(channelID, projectionKey1)
	require.NoError(err)
	err = broker.Subscribe(channelID, projectionKey1)
	require.NoError(err)

	// check metrics - must be 1, not 2
	require.Equal(1, broker.MetricNumSubscriptions())
	broker.MetricSubject(watchCtx, func(subject istructs.SubjectLogin, numChannels, numSubscriptions int) {
		require.Equal(subject, testSubject)
		require.Equal(1, numChannels)
		require.Equal(1, numSubscriptions)
	})

	// double unsubscribe
	require.NoError(broker.Unsubscribe(channelID, projectionKey1))
	require.NoError(broker.Unsubscribe(channelID, projectionKey1))

	cancel()
	wg.Wait()
	channelCleanup()
	checkMetricsZero(t, broker, projectionKey1)
	brokerCleanup()
}
