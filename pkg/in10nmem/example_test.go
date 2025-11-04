/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 * @author Maxim Geraskin
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package in10nmem_test

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/goutils/timeu"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/in10nmem"
	istructs "github.com/voedger/voedger/pkg/istructs"
)

func Example() {

	logger.SetLogLevel(logger.LogLevelNone)

	var wg sync.WaitGroup
	c := new(callbackMock)
	c.data = make(chan UpdateUnit)

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
	ctx, cancel := context.WithCancel(context.Background())

	broker, cleanup := in10nmem.NewN10nBroker(quotasExample, timeu.NewITime())
	defer cleanup()

	numChannels := broker.MetricNumChannels()
	fmt.Println("Before NewChannel(), numChannels:", numChannels)

	const subject istructs.SubjectLogin = "paa"

	// Create new channel

	channel, channelCleanup, err := broker.NewChannel(subject, 24*time.Hour)
	checkTrue(err == nil, err)
	numChannels = broker.MetricNumChannels()
	fmt.Println("After NewChannel(), numChannels:", numChannels)

	// Run a watcher

	wg.Add(1)
	go func() {
		defer wg.Done()
		broker.WatchChannel(ctx, channel, c.updatesMock)
	}()

	// check subscriptions, numSubscriptions must be equal 0
	fmt.Println("Before Subscribe(), numSubscriptions: ", broker.MetricNumSubscriptions())

	// Subscribe on exist channel numSubscriptions must be equal 1
	err = broker.Subscribe(channel, projectionKeyExample)
	if err != nil {
		panic(err)
	}

	fmt.Println("Before Subscribe(), numSubscriptions: ", broker.MetricNumSubscriptions())

	// Update the projection

	broker.Update(projectionKeyExample, istructs.Offset(122))
	broker.Update(projectionKeyExample, istructs.Offset(123))
	broker.Update(projectionKeyExample, istructs.Offset(124))
	broker.Update(projectionKeyExample, istructs.Offset(125))
	broker.Update(projectionKeyExample, istructs.Offset(126))

	// Wait until last update will be processed
	for update := range c.data {
		if update.Offset == istructs.Offset(126) {
			break
		}
	}

	// Cancel the watcher
	cancel()

	// Wait until the watcher will be finished
	wg.Wait()

	// finalize the channel
	channelCleanup()

	// Check subscriptions, numSubscriptions must be equal 0
	fmt.Println("Canceled, numSubscriptions: ", broker.MetricNumSubscriptions())

	// Output:
	// Before NewChannel(), numChannels: 0
	// After NewChannel(), numChannels: 1
	// Before Subscribe(), numSubscriptions:  0
	// Before Subscribe(), numSubscriptions:  1
	// Canceled, numSubscriptions:  0

}

type UpdateUnit struct {
	Projection in10n.ProjectionKey
	Offset     istructs.Offset
}

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

func checkTrue(value bool, info ...interface{}) {
	if !value {
		panic(fmt.Sprint(info...))
	}
}
