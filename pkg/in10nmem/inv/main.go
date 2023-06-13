/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/in10nmem"
	in10nmemv1 "github.com/voedger/voedger/pkg/in10nmem/v1"
	in10nmemv3 "github.com/voedger/voedger/pkg/in10nmem/v3"
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

	if len(os.Args) < 2 {
		println("Use v1 or v2 as argument")
		return
	}

	if os.Args[1] == "v1" {
		println("Running v1...")
		nb := in10nmemv1.Provide(quotas)
		runChannels(nb)
	}
	if os.Args[1] == "v2" {
		println("Running v2...")
		nb, cleanup := in10nmem.ProvideEx2(quotas, time.Now)
		defer cleanup()

		runChannels(nb)
	}
	if os.Args[1] == "v3" {
		println("Running v3...")
		nb, cleanup := in10nmemv3.ProvideEx2(quotas, time.Now)
		defer cleanup()

		runChannels(nb)
	}
	log.Fatal("Unknown argument", os.Args[1])

}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

const numAttackers = 500
const numPartitions = 10000
const numProjectorsPerPartition = 100
const eventsPerSeconds = 100
const subject istructs.SubjectLogin = "main"

var projectionPLog = appdef.NewQName("sys", "plog")

func runChannels(broker in10n.IN10nBroker) {

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

	println("numAttackers: ", numAttackers)
	println("numPartitions: ", numPartitions)
	println("numProjectorsPerPartition: ", numProjectorsPerPartition)
	println("eventsPerSeconds: ", eventsPerSeconds)

	var wrkCount int64
	var wrkSumLatenciesNano int64
	var wrkOffset int64 = 0

	for i := 0; i < numAttackers; i++ {
		go func() {
			t := time.NewTicker(1 * time.Second / eventsPerSeconds)

			// nolint
			partition := rand.Intn(numPartitions)

			projectionKeyExample := in10n.ProjectionKey{
				App:        istructs.AppQName_test1_app1,
				Projection: projectionPLog,
				WS:         istructs.WSID(partition),
			}

			for range t.C {
				newOffset := atomic.AddInt64(&wrkOffset, 1)
				updateTime := time.Now()
				broker.Update(projectionKeyExample, istructs.Offset(newOffset))
				atomic.AddInt64(&wrkSumLatenciesNano, int64(time.Since(updateTime).Nanoseconds()))
				atomic.AddInt64(&wrkCount, 1)
			}
		}()
	}

	t := time.NewTicker(1 * time.Second)
	startTime := time.Now()
	for range t.C {
		count := atomic.LoadInt64(&wrkCount)
		sumLatenciesNano := atomic.LoadInt64(&wrkSumLatenciesNano)
		fmt.Println("count: ", count,
			"sumLatenciesNano: ", sumLatenciesNano,
			"rps:", float64(count)/float64(time.Since(startTime).Seconds()),
			"avg. latency, ns:", float64(sumLatenciesNano)/float64(count),
		)

	}

	wg.Wait()

}

func runChannel(channelID in10n.ChannelID, broker in10n.IN10nBroker) {

	broker.WatchChannel(context.Background(), channelID, updatesMock)

}

func updatesMock(projection in10n.ProjectionKey, offset istructs.Offset) {
	time.Sleep(1 * time.Millisecond)

}
