/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package main

import (
	"context"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/timeu"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/in10nmem"
	in10nmemv1 "github.com/voedger/voedger/pkg/in10nmem/v1"
	"github.com/voedger/voedger/pkg/istructs"
)

func main() {

	const BigNumber = 1000000000000000000

	quotas := in10n.Quotas{
		Channels:                BigNumber,
		ChannelsPerSubject:      BigNumber,
		Subscriptions:           BigNumber,
		SubscriptionsPerSubject: BigNumber,
	}

	if len(os.Args) < 2 {
		println("Use v1 or v2 as argument")
		return
	}

	switch os.Args[1] {
	case "v1":
		println("Running v1...")
		nb := in10nmemv1.Provide(quotas)
		runChannels(nb)
	case "v2":
		println("Running v2...")
		nb, cleanup := in10nmem.ProvideEx2(quotas, timeu.NewITime())
		defer cleanup()

		runChannels(nb)
	default:
		log.Fatal("Unknown argument", os.Args[1])
	}
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

const numPartitions = 200
const numProjectorsPerPartition = 1000
const eventsPerSeconds = 500
const subject istructs.SubjectLogin = "main"

var projectionPLog = appdef.NewQName("sys", "plog")

func runChannels(broker in10n.IN10nBroker) {

	wg := sync.WaitGroup{}

	for partition := 0; partition < numPartitions; partition++ {

		for projector := 0; projector < numProjectorsPerPartition; projector++ {

			channelID, err := broker.NewChannel(subject, hours24)
			checkErr(err)

			projectionKeyExample := in10n.ProjectionKey{
				App:        istructs.AppQName_test1_app1,
				Projection: projectionPLog,
				WS:         istructs.WSID(partition),
			}

			err = broker.Subscribe(channelID, projectionKeyExample)
			checkErr(err)

			wg.Add(1)
			go runChannel(channelID, broker)
		}
	}

	t := time.NewTicker(1 * time.Second / eventsPerSeconds)

	println("numPartitions: ", numPartitions)
	println("numProjectorsPerPartition: ", numProjectorsPerPartition)
	println("eventsPerSeconds: ", eventsPerSeconds)

	offset := 0
	for range t.C {

		// nolint
		partition := rand.Intn(numPartitions)

		projectionKeyExample := in10n.ProjectionKey{
			App:        istructs.AppQName_test1_app1,
			Projection: projectionPLog,
			WS:         istructs.WSID(partition), // nolint G115
		}
		broker.Update(projectionKeyExample, 0)
		offset++

	}

	wg.Wait()

}

func runChannel(channelID in10n.ChannelID, broker in10n.IN10nBroker) {

	broker.WatchChannel(context.Background(), channelID, updatesMock)

}

func updatesMock(projection in10n.ProjectionKey, offset istructs.Offset) {

}
