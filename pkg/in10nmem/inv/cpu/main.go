package main

import (
	"fmt"
	"time"

	"github.com/shirou/gopsutil/cpu"
)

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
	return (totalDelta - idleDelta) / totalDelta * 100, nil // returning usage in percentage
}

// getCPUTotalTime calculates the total time spent by CPU
func getCPUTotalTime(t cpu.TimesStat) float64 {
	return t.User + t.System + t.Idle + t.Nice + t.Iowait + t.Irq + t.Softirq + t.Steal
}

func main() {
	data, err := initCPUUsage()
	if err != nil {
		fmt.Println("Error initializing CPU usage:", err)
		return
	}

	for {
		usage, err := calcCPUsage(&data)
		if err != nil {
			fmt.Println("Error calculating CPU usage:", err)
			return
		}

		fmt.Printf("CPU Usage: %.2f%%\n", usage)
		time.Sleep(time.Second)
	}
}