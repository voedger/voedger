/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/spf13/cobra"
	"github.com/untillpro/goutils/logger"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/in10nmem"
	in10nmemv1 "github.com/voedger/voedger/pkg/in10nmem/v1"
	in10nmemv3 "github.com/voedger/voedger/pkg/in10nmem/v3"
	"github.com/voedger/voedger/pkg/istructs"

	"github.com/shirou/gopsutil/cpu"
)

var numPartitions int
var numProjectorsPerPartition int
var v2NumNotifiers int
var verbose bool
var csv bool

var numN10nLock sync.Mutex
var partitionNumN10ns = make(map[int]int64)
var numN10ns int64

func main() {
	rootCmd := &cobra.Command{
		Use:   "go run main.go v1/v2/v3",
		Short: "",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if verbose {
				logger.SetLogLevel(logger.LogLevelVerbose)
			}
		},
	}

	// nolint
	{
		rootCmd.PersistentFlags().IntVar(&numPartitions, "numPartitions", 1000, "Number of partitions")
		rootCmd.PersistentFlags().BoolVar(&csv, "csv", false, "Use comma-separated values output")
		rootCmd.PersistentFlags().IntVar(&numProjectorsPerPartition, "numProjectorsPerPartition", 1000, "Number of projectors per partition")
		rootCmd.PersistentFlags().IntVar(&v2NumNotifiers, "v2NumNotifiers", 1, "Number of v2 notifiers")
		rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose mode")
	}

	const BigNumber = 1000000000000000000
	quotas := in10n.Quotas{
		Channels:               BigNumber,
		ChannelsPerSubject:     BigNumber,
		Subsciptions:           BigNumber,
		SubsciptionsPerSubject: BigNumber,
	}

	v1Cmd := &cobra.Command{
		Use:   "v1",
		Short: "Run notifier.v1",
		Run: func(cmd *cobra.Command, args []string) {
			println("Running v1...")
			nb := in10nmemv1.Provide(quotas)
			runChannels(cmd.Use, nb)
		},
	}

	v2Cmd := &cobra.Command{
		Use:   "v2",
		Short: "Run notifier.v2",
		Run: func(cmd *cobra.Command, args []string) {
			println("Running v2...")
			nb, cleanup := in10nmem.ProvideEx3(quotas, time.Now, v2NumNotifiers)
			defer cleanup()
			runChannels(cmd.Use, nb)
		},
	}

	v3Cmd := &cobra.Command{
		Use:   "v3",
		Short: "Run notifier.v3",
		Run: func(cmd *cobra.Command, args []string) {
			println("Running v3...")
			nb, cleanup := in10nmemv3.ProvideEx2(quotas, time.Now)
			defer cleanup()
			runChannels(cmd.Use, nb)
		},
	}

	rootCmd.AddCommand(v1Cmd, v2Cmd, v3Cmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

const numAttackers = 1
const eventsPerSeconds = 100000 // Just a very big number, doesn't matter for one attacker
const subject istructs.SubjectLogin = "main"

var projectionPLog = appdef.NewQName("sys", "plog")

func runChannels(version string, broker in10n.IN10nBroker) {

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

	println("numPartitions: ", numPartitions)
	println("numProjectorsPerPartition: ", numProjectorsPerPartition)
	println("v2NumNotifiers: ", v2NumNotifiers)

	println("numAttackers: ", numAttackers)
	println("eventsPerSeconds: ", eventsPerSeconds)

	var updateCount int64
	var updateSumLatenciesNano int64
	var updateOffset int64 = 0

	for i := 0; i < numAttackers; i++ {
		go func() {
			t := time.NewTicker(1 * time.Second / eventsPerSeconds)

			projectionKeyExample := in10n.ProjectionKey{
				App:        istructs.AppQName_test1_app1,
				Projection: projectionPLog,
			}

			for range t.C {

				// nolint
				partition := rand.Intn(numPartitions)
				projectionKeyExample.WS = istructs.WSID(partition)

				newOffset := atomic.AddInt64(&updateOffset, 1)
				updateTime := time.Now()
				broker.Update(projectionKeyExample, istructs.Offset(newOffset))
				atomic.AddInt64(&updateSumLatenciesNano, int64(time.Since(updateTime).Nanoseconds()))
				atomic.AddInt64(&updateCount, 1)
			}
		}()
	}

	t := time.NewTicker(5 * time.Second)
	startTime := time.Now()
	startCPU, _ := initCPUUsage()
	measures := 0
	var sumRAM uint64 = 0
	if csv {
		fmt.Println("Version, PartsxProjections, RPS, Latency, CPU, avgRAM, N10n Rate")
	}
	for range t.C {
		measures++
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)

		count := atomic.LoadInt64(&updateCount)
		sumLatenciesNano := atomic.LoadInt64(&updateSumLatenciesNano)
		CPU, _ := calcCPUsage2(startCPU)
		RAM := memStats.Alloc
		sumRAM += RAM

		rps := float64(count) / float64(time.Since(startTime).Seconds())
		latency := float64(sumLatenciesNano) / float64(count)

		//nolint
		n10rate := float64(numN10ns) / float64(count*int64(numProjectorsPerPartition)) * 100
		//nolint
		avgRAM := float64(sumRAM) / float64(measures) / 1024 / 1024
		if !csv {
			fmt.Println(
				"rps:", rps,
				"latency, ns:", latency,
				"n10n rate,%:", n10rate,
				"CPU, %:", CPU*100,
				"avg(RAM), MB:", avgRAM,
				"RAM, MB:", memStats.Alloc/1024/1024,
			)
		} else {
			//nolint
			fmt.Printf("%s| %dx%d| %.2f| %v| %.2f%%| %.0fMB| %.2f%%\n",
				version, numPartitions, numProjectorsPerPartition, rps, time.Duration(latency), CPU*100, avgRAM, n10rate,
			)
		}

		if logger.IsVerbose() {
			numN10nLock.Lock()
			logger.Verbose("partitionN10ns: ", partitionNumN10ns)
			numN10nLock.Unlock()
		}

	}
	wg.Wait()

}

func runChannel(channelID in10n.ChannelID, broker in10n.IN10nBroker) {

	broker.WatchChannel(context.Background(), channelID, notifySubscriber)

}

func notifySubscriber(projection in10n.ProjectionKey, offset istructs.Offset) {
	numN10nLock.Lock()
	defer numN10nLock.Unlock()
	partitionNumN10ns[int(projection.WS)]++
	numN10ns++
}

// CPUUsageData struct will hold the necessary data for CPU usage calculation
type CPUUsageData struct {
	lastTotalTime float64
	lastIdleTime  float64
}

// initCPUUsage initializes and returns CPU usage data structure
func initCPUUsage() (CPUUsageData, error) {
	times, err := cpu.Times(false) // false for total system, not per-CPU stats
	if err != nil {
		return CPUUsageData{}, err
	}

	totalTime := getCPUTotalTime(times[0])
	idleTime := times[0].Idle
	return CPUUsageData{lastTotalTime: totalTime, lastIdleTime: idleTime}, nil
}

// calcCPUsage calculates and returns the CPU usage since the first calculation
func calcCPUsage2(firstData CPUUsageData) (float64, error) {
	times, err := cpu.Times(false) // false for total system, not per-CPU stats
	if err != nil {
		return 0, err
	}

	totalTime := getCPUTotalTime(times[0])
	idleTime := times[0].Idle

	totalDelta := totalTime - firstData.lastTotalTime
	idleDelta := idleTime - firstData.lastIdleTime

	if totalDelta == 0 {
		return 0, nil
	}
	return (totalDelta - idleDelta) / totalDelta, nil // returning usage in percentage
}

// nolint
// calcCPUsage calculates and returns the CPU usage since last calculation
func calcCPUsage(data *CPUUsageData) (float64, error) {
	times, err := cpu.Times(false) // false for total system, not per-CPU stats
	if err != nil {
		return 0, err
	}

	totalTime := getCPUTotalTime(times[0])
	idleTime := times[0].Idle

	totalDelta := totalTime - data.lastTotalTime
	idleDelta := idleTime - data.lastIdleTime

	data.lastTotalTime = totalTime
	data.lastIdleTime = idleTime

	if totalDelta == 0 {
		return 0, nil
	}
	return (totalDelta - idleDelta) / totalDelta, nil // returning usage in percentage
}

// getCPUTotalTime calculates the total time spent by CPU
func getCPUTotalTime(t cpu.TimesStat) float64 {
	return t.User + t.System + t.Idle + t.Nice + t.Iowait + t.Irq + t.Softirq + t.Steal
}
