/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package main

import (
	"fmt"
	"time"

	"github.com/shirou/gopsutil/cpu"
)

func main() {

	_, _ = cpu.Percent(0, true) // for init

	for {
		usage, err := cpu.Percent(0, true)
		if err != nil {
			fmt.Println("Error calculating CPU usage:", err)
			return
		}
		fmt.Printf("CPU Usage: %.2f%%\n", usage)
		time.Sleep(time.Second)
	}
}
