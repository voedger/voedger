/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */

package iextenginewasm

import (
	"context"
	"testing"

	"github.com/voedger/voedger/pkg/iextengine"
)

func bench_purecall(b *testing.B) {
	ctx := context.Background()

	moduleUrl := testModuleURL("./_testdata/allocs/pkg.wasm")
	ee, closer, err := ExtEngineWazeroFactory(ctx, moduleUrl, iextengine.ExtEngineConfig{MemoryLimitPages: 0xffff})
	if err != nil {
		panic(err)
	}
	extensions := extractExtensions(ee)
	ee.SetLimits(limits)
	simple := extensions["simple"]

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		if e := simple.Invoke(extIO); e != nil {
			panic(e)
		}
	}
	b.StopTimer()
	closer()
}

func bench_gc(b *testing.B, cycles int) {

	ctx := context.Background()
	moduleUrl := testModuleURL("./_testdata/allocs/pkggc.wasm")
	ee, closer, err := ExtEngineWazeroFactory(ctx, moduleUrl, iextengine.ExtEngineConfig{MemoryLimitPages: 0xffff})
	if err != nil {
		panic(err)
	}
	extensions := extractExtensions(ee)
	arrAppend := extensions["arrAppend"]
	arrReset := extensions["arrReset"]
	ee.SetLimits(limits)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		b.StopTimer()
		for i := 0; i < cycles; i++ {
			if e := arrAppend.Invoke(extIO); e != nil {
				panic(e)
			}
		}
		if e := arrReset.Invoke(extIO); e != nil {
			panic(e)
		}
		b.StartTimer()
		if e := ee.(*wazeroExtEngine).gc(); e != nil {
			panic(e)
		}
	}

	b.StopTimer()
	closer()

}

/*
goos: linux
goarch: amd64
cpu: 12th Gen Intel(R) Core(TM) i7-12700
Benchmark_GarbageCollection/simple-call-no-gc-20         	12560898	        82.73 ns/op	       0 B/op	       0 allocs/op
Benchmark_GarbageCollection/gc-after-no-allocs-20        	      24	  47296163 ns/op	     888 B/op	      12 allocs/op
Benchmark_GarbageCollection/gc-after-6-allocs-48b-20     	      25	  47137050 ns/op	     888 B/op	      12 allocs/op
Benchmark_GarbageCollection/gc-after-20000-allocs-20     	      24	  46626752 ns/op	     888 B/op	      12 allocs/op
*/
func Benchmark_GarbageCollection(b *testing.B) {
	b.Run("simple-call-no-gc", func(b *testing.B) {
		bench_purecall(b)
	})
	b.Run("gc-after-no-allocs", func(b *testing.B) {
		bench_gc(b, 0)
	})
	b.Run("gc-after-6-allocs-48b", func(b *testing.B) {
		bench_gc(b, 3)
	})
	b.Run("gc-after-20000-allocs", func(b *testing.B) {
		bench_gc(b, 10000)
	})
}

func bench_extensions(b *testing.B, gc bool) {

	ctx := context.Background()
	wsm := "./_testdata/benchmarks/pkg.wasm"
	if gc {
		wsm = "./_testdata/benchmarks/pkggc.wasm"
	}
	moduleUrl := testModuleURL(wsm)
	ee, closer, err := ExtEngineWazeroFactory(ctx, moduleUrl, iextengine.ExtEngineConfig{MemoryLimitPages: 0xffff})
	if err != nil {
		panic(err)
	}
	defer closer()
	extensions := extractExtensions(ee)

	testfunc := func(extname string) {
		extension := extensions[extname]
		b.Run(extname, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				err := extension.Invoke(extIO)
				if err != nil {
					panic(err)
				}
			}
		})
	}

	testfunc("oneGetOneIntent5calls")
	testfunc("oneGetNoIntents2calls")
	testfunc("oneGetLongStr3calls")
	testfunc("oneKey1call")
	testfunc("doNothing")
}

/*
goos: linux
goarch: amd64
cpu: 12th Gen Intel(R) Core(TM) i7-12700
Benchmark_Extensions_NoGc/oneGetOneIntent5calls-20       	  830360	      1257 ns/op	    2119 B/op	      25 allocs/op
Benchmark_Extensions_NoGc/oneGetNoIntents2calls-20       	 1795513	       641.7 ns/op	    1248 B/op	      13 allocs/op
Benchmark_Extensions_NoGc/oneGetLongStr3calls-20         	 1735144	       669.2 ns/op	    1368 B/op	      13 allocs/op
Benchmark_Extensions_NoGc/oneKey1call-20                 	 4694110	       243.9 ns/op	     280 B/op	       5 allocs/op
Benchmark_Extensions_NoGc/doNothing-20                   	19467026	        61.33 ns/op	       0 B/op	       0 allocs/op
*/
func Benchmark_Extensions_NoGc(b *testing.B) {
	bench_extensions(b, false)
}
func Skip_Benchmark_Extensions_WithGc(b *testing.B) {
	bench_extensions(b, true)
}

func benchmarkRecover(b *testing.B, limitPages uint, expectedRuns int) {
	ctx := context.Background()
	moduleUrl := testModuleURL("./_testdata/allocs/pkg.wasm")
	ee, closer, err := ExtEngineWazeroFactory(ctx, moduleUrl, iextengine.ExtEngineConfig{MemoryLimitPages: limitPages})
	if err != nil {
		panic(err)
	}
	defer closer()
	extensions := extractExtensions(ee)
	extAppend := extensions["arrAppend2"]
	we := ee.(*wazeroExtEngine)

	for runs := 0; runs < expectedRuns; runs++ {
		if err := extAppend.Invoke(extIO); err != nil {
			panic(err)
		}
	}

	memoryFull := we.module.Memory().Backup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StartTimer()
		we.recover()
		// if extAppend.Invoke(extIO) == nil {
		// 	panic("err expected")
		// }
		//require.Equal(b, uint64(0x6ebc50), h)
		b.StopTimer()
		we.module.Memory().Restore(memoryFull)
	}
}

func benchmarkRecoverClean(b *testing.B, limitPages uint) {
	ctx := context.Background()
	moduleUrl := testModuleURL("./_testdata/allocs/pkg.wasm")
	ee, closer, err := ExtEngineWazeroFactory(ctx, moduleUrl, iextengine.ExtEngineConfig{MemoryLimitPages: limitPages})
	if err != nil {
		panic(err)
	}
	defer closer()
	we := ee.(*wazeroExtEngine)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		we.recover()
	}
}

/*
goos: linux
goarch: amd64
pkg: github.com/heeus/core/iextenginewazero
cpu: 12th Gen Intel(R) Core(TM) i7-12700
Benchmark_Recover/2Mib-1%-20         	  491917	      2041 ns/op	       0 B/op	       0 allocs/op
Benchmark_Recover/2Mib-50%-20        	   17457	     68422 ns/op	       0 B/op	       0 allocs/op
Benchmark_Recover/2Mib-100%-20       	   18838	     64025 ns/op	       0 B/op	       0 allocs/op
Benchmark_Recover/8Mib-100%-20       	    5707	    204310 ns/op	       7 B/op	       0 allocs/op
Benchmark_Recover/100Mib-70%-20      	    6247	    192577 ns/op	      12 B/op	       0 allocs/op
*/
func Benchmark_Recover(b *testing.B) {
	WasmPreallocatedBufferSize = 20000
	b.Run("2Mib-1%", func(b *testing.B) { benchmarkRecoverClean(b, 0x20) })
	WasmPreallocatedBufferSize = 1000000
	b.Run("2Mib-50%", func(b *testing.B) { benchmarkRecoverClean(b, 0x20) })
	b.Run("2Mib-100%", func(b *testing.B) { benchmarkRecover(b, 0x20, 3) })
	b.Run("8Mib-100%", func(b *testing.B) { benchmarkRecover(b, 0x80, 26) })
	b.Run("100Mib-70%", func(b *testing.B) { benchmarkRecover(b, 0x5f5, 209) })
}

func Benchmark_ArrayCopy(b *testing.B) {
	const backupSize = 2000000
	const heapSize = 10000000
	backup := make([]byte, backupSize)
	heap := make([]byte, heapSize)
	_ = append(heap, 1)
	b.Run("recommended", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			//_ = append(heap[0:0], backup...)
			heap = make([]byte, len(backup))
			copy(heap, backup)
			b.StopTimer()
			heap = make([]byte, heapSize)
			_ = append(heap, 1)
			b.StartTimer()
		}
	})
	b.Run("shrink", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			heap = heap[0:len(backup)]
			copy(heap[0:len(backup)], backup[0:])
			b.StopTimer()
			heap = make([]byte, heapSize)
			_ = append(heap, 1)
			b.StartTimer()
		}
	})

}
