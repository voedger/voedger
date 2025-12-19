// Copyright (c) 2021-present Voedger Authors.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.
// @author Maxim Geraskin

package pipeline

import (
	"context"
	"strconv"
	"testing"
	"time"
)

func newBenchPipeline(nops int) ISyncPipeline {
	ops := make([]*WiredOperator, nops)
	for idx := range ops {
		ops[idx] = WireFunc[IWorkpiece](strconv.Itoa(idx), nil)
	}

	pipeline := NewSyncPipeline(context.Background(), "bench 10 NOOPS", ops[0], ops[1:]...)
	return pipeline
}

func Benchmark_10_NOPS(b *testing.B) {

	pipeline := newBenchPipeline(10)
	start := time.Now()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = pipeline.SendSync(noRelease{})
	}

	elapsed := time.Since(start).Seconds()
	b.ReportMetric(float64(b.N)/elapsed, "rps")
}

func Benchmark_100_NOPS(b *testing.B) {

	pipeline := newBenchPipeline(100)
	start := time.Now()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = pipeline.SendSync(noRelease{})
	}

	b.ReportMetric(float64(b.N)/time.Since(start).Seconds(), "rps")
}

func Benchmark_10_NOPS_Parallel(b *testing.B) {

	pipeline := newBenchPipeline(10)
	start := time.Now()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = pipeline.SendSync(noRelease{})
		}
	})

	b.ReportMetric(float64(b.N)/time.Since(start).Seconds(), "rps")
}

func Benchmark_100_NOPS_Parallel(b *testing.B) {

	pipeline := newBenchPipeline(100)
	start := time.Now()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = pipeline.SendSync(noRelease{})
		}
	})
	b.ReportMetric(float64(b.N)/time.Since(start).Seconds(), "rps")
}
