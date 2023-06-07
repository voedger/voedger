package main

import (
	"context"
	"math/rand"
	"runtime"
	"sync"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/in10n"
	in10nmemv1 "github.com/voedger/voedger/pkg/in10nmem/v1"
	"github.com/voedger/voedger/pkg/istructs"
)

func main() {

	const BigNumber = 1000000000000000000

	quotas := in10n.Quotas{
		Channels:               BigNumber,
		ChannelsPerSubject:     BigNumber,
		Subsciptions:           BigNumber,
		SubsciptionsPerSubject: BigNumber,
	}

	nb := in10nmemv1.Provide(quotas)
	runChannels(nb)

}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

const numCores = 4
const numPartitions = 200
const numProjectorsPerPartition = 500
const eventsPerSeconds = 1
const subject istructs.SubjectLogin = "main"

var projectionPLog = appdef.NewQName("sys", "plog")

func runChannels(broker in10n.IN10nBroker) {

	runtime.GOMAXPROCS(numCores)

	wg := sync.WaitGroup{}

	for partition := 0; partition < numPartitions; partition++ {

		for projector := 0; projector < numProjectorsPerPartition; projector++ {

			channelID, err := broker.NewChannel(subject, 24*time.Hour)
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

	println("numCores: ", numCores)
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
			WS:         istructs.WSID(partition),
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
