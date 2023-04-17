/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package ibusmem

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/voedger/voedger/pkg/ibus"
)

func Benchmark_QueryAndSend(b *testing.B) {
	// maximum 10 concurrent requests, rwTimeout = 1 second
	busimpl, cleanup := New(ibus.CLIParams{MaxNumOfConcurrentRequests: 10, ReadWriteTimeout: time.Second * 1})
	defer cleanup()

	// EchoReceiver, two reading goroutines, channel buffer is 10
	busimpl.RegisterReceiver("owner", "app", 0, "q", ibus.EchoReceiver, 1, 10)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sender, ok := busimpl.QuerySender("owner", "app", 0, "q")
		if !ok {
			panic("sender not found")
		}
		_, _, err := sender.Send(ctx, "hello123", ibus.NullHandler)
		if err != nil {
			panic(err)
		}
	}
	b.StopTimer()
}

func Benchmark_Send(b *testing.B) {
	busimpl, cleanup := New(ibus.CLIParams{MaxNumOfConcurrentRequests: 10, ReadWriteTimeout: time.Second * 1})
	defer cleanup()

	// EchoReceiver, two reading goroutines, channel buffer is 10
	busimpl.RegisterReceiver("owner", "app", 0, "q", ibus.EchoReceiver, 1, 10)

	ctx := context.Background()

	sender, ok := busimpl.QuerySender("owner", "app", 0, "q")
	if !ok {
		panic("sender not found")
	}

	for i := 0; i < b.N; i++ {
		_, _, err := sender.Send(ctx, "hello123", ibus.NullHandler)
		if err != nil {
			panic(err)
		}
	}
}

func Benchmark_10Sections(b *testing.B) {
	busimpl, cleanup := New(ibus.CLIParams{MaxNumOfConcurrentRequests: 10, ReadWriteTimeout: time.Second * 1})
	defer cleanup()

	busimpl.RegisterReceiver("owner", "app", 0, "q", responseWithSections, 1, 1)

	ctx := context.Background()

	sender, ok := busimpl.QuerySender("owner", "app", 0, "q")
	if !ok {
		panic("sender not found")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := sender.Send(ctx, 10, ibus.NullHandler)
		if err != nil {
			panic(err)
		}
	}
	b.StopTimer()
}

// TODO: why  3181 ns/op
func Benchmark_QS_Parallel(b *testing.B) {

	busimpl, cleanup := New(ibus.CLIParams{MaxNumOfConcurrentRequests: 10, ReadWriteTimeout: time.Second * 1})
	defer cleanup()

	cpu := runtime.NumCPU()
	busimpl.RegisterReceiver("owner", "app", 0, "q", ibus.EchoReceiver, cpu, 10)
	ctx := context.Background()

	b.RunParallel(func(pb *testing.PB) {
		sender, ok := busimpl.QuerySender("owner", "app", 0, "q")
		if !ok {
			panic("sender not found")
		}
		for pb.Next() {
			_, _, err := sender.Send(ctx, "hello123", ibus.NullHandler)
			if err != nil {
				panic(err)
			}
		}
	})
	b.StopTimer()
}
