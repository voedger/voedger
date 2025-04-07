/*
  - Copyright (c) 2023-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/

package iextenginewazero

import (
	"context"
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iextengine"
)

func bench_purecall(b *testing.B) {
	ctx := context.Background()
	const simple = "simple"
	moduleURL := testModuleURL("./_testdata/allocs/pkg.wasm")
	ee, err := testFactoryHelper(ctx, moduleURL, []string{simple}, iextengine.ExtEngineConfig{MemoryLimitPages: 0xffff}, false)
	if err != nil {
		panic(err)
	}
	//ee.SetLimits(limits)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		if e := ee.Invoke(context.Background(), appdef.NewFullQName("test", simple), extIO); e != nil {
			panic(e)
		}
	}
	b.StopTimer()
	ee.Close(ctx)
}

func bench_gc(b *testing.B, cycles int) {

	const arrAppend = "arrAppend"
	const arrReset = "arrReset"
	ctx := context.Background()
	moduleURL := testModuleURL("./_testdata/allocs/pkggc.wasm")
	ee, err := testFactoryHelper(ctx, moduleURL, []string{arrAppend, arrReset}, iextengine.ExtEngineConfig{MemoryLimitPages: 0xffff}, false)
	if err != nil {
		panic(err)
	}
	//ee.SetLimits(limits)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		b.StopTimer()
		for i := 0; i < cycles; i++ {
			if e := ee.Invoke(context.Background(), appdef.NewFullQName("test", arrAppend), extIO); e != nil {
				panic(e)
			}
		}
		if e := ee.Invoke(context.Background(), appdef.NewFullQName("test", arrReset), extIO); e != nil {
			panic(e)
		}
		b.StartTimer()
		if e := ee.(*wazeroExtEngine).gc(testPkg, ctx); e != nil {
			panic(e)
		}
	}

	b.StopTimer()
	ee.Close(ctx)

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

func bench_extensions(b *testing.B, gc bool, compile bool) {

	funcs := []string{"oneGetOneIntent5calls", "oneGetNoIntents2calls", "oneGetLongStr3calls", "oneKey1call", "doNothing"}

	ctx := context.Background()
	wsm := "./_testdata/benchmarks/pkg.wasm"
	if gc {
		wsm = "./_testdata/benchmarks/pkggc.wasm"
	}
	moduleURL := testModuleURL(wsm)
	ee, err := testFactoryHelper(ctx, moduleURL, funcs, iextengine.ExtEngineConfig{MemoryLimitPages: 0xffff}, compile)
	if err != nil {
		panic(err)
	}
	defer ee.Close(ctx)
	for _, extname := range funcs {
		ext := appdef.NewFullQName(testPkg, extname)
		b.Run(extname, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				err := ee.Invoke(context.Background(), ext, extIO)
				if err != nil {
					panic(err)
				}
			}
		})
	}
}

/*
goos: linux
goarch: amd64
cpu: 12th Gen Intel(R) Core(TM) i7-12700
Benchmark_Extensions_NoGc/Compiler/oneGetOneIntent5calls-20         	  307974	      5875 ns/op	    2988 B/op	      55 allocs/op
Benchmark_Extensions_NoGc/Compiler/oneGetNoIntents2calls-20         	  728787	      2519 ns/op	    1728 B/op	      26 allocs/op
Benchmark_Extensions_NoGc/Compiler/oneGetLongStr3calls-20           	   90838	     48478 ns/op	  133032 B/op	      33 allocs/op
Benchmark_Extensions_NoGc/Compiler/oneKey1call-20                   	 1000000	      1449 ns/op	     640 B/op	      15 allocs/op
Benchmark_Extensions_NoGc/Compiler/doNothing-20                     	 3906926	       303.5 ns/op	     160 B/op	       3 allocs/op
Benchmark_Extensions_NoGc/Interpreter/oneGetOneIntent5calls-20      	  267399	      5619 ns/op	    3156 B/op	      61 allocs/op
Benchmark_Extensions_NoGc/Interpreter/oneGetNoIntents2calls-20      	  628476	      2079 ns/op	    1800 B/op	      29 allocs/op
Benchmark_Extensions_NoGc/Interpreter/oneGetLongStr3calls-20        	   31657	     43928 ns/op	  133128 B/op	      37 allocs/op
Benchmark_Extensions_NoGc/Interpreter/oneKey1call-20                	  924883	      1109 ns/op	     688 B/op	      17 allocs/op
Benchmark_Extensions_NoGc/Interpreter/doNothing-20                  	 3429517	       327.5 ns/op	     184 B/op	       4 allocs/op
*/
func Benchmark_Extensions_NoGc(b *testing.B) {
	b.Run("Compiler", func(b *testing.B) {
		bench_extensions(b, false, true)
	})
	b.Run("Interpreter", func(b *testing.B) {
		bench_extensions(b, false, false)
	})
}
func Skip_Benchmark_Extensions_WithGc(b *testing.B) {
	bench_extensions(b, true, true)
}

func benchmarkRecover(b *testing.B, limitPages uint, expectedRuns int) {
	const arrAppend2 = "arrAppend2"
	ctx := context.Background()
	moduleURL := testModuleURL("./_testdata/allocs/pkg.wasm")
	ee, err := testFactoryHelper(ctx, moduleURL, []string{arrAppend2}, iextengine.ExtEngineConfig{MemoryLimitPages: limitPages}, true)
	if err != nil {
		panic(err)
	}
	defer ee.Close(ctx)

	we := ee.(*wazeroExtEngine)
	we.autoRecover = false

	ext := appdef.NewFullQName(testPkg, arrAppend2)
	for runs := 0; runs < expectedRuns; runs++ {
		if err := ee.Invoke(context.Background(), ext, extIO); err != nil {
			panic(err)
		}
	}

	//we.backupMemory()

	// the next call should fail
	if err := ee.Invoke(context.Background(), ext, extIO); err == nil {
		panic("err expected")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		we.recover(context.Background())
	}
}

func benchmarkRecoverClean(b *testing.B, limitPages uint) {
	ctx := context.Background()
	moduleURL := testModuleURL("./_testdata/allocs/pkg.wasm")
	ee, err := testFactoryHelper(ctx, moduleURL, []string{}, iextengine.ExtEngineConfig{MemoryLimitPages: limitPages}, true)
	if err != nil {
		panic(err)
	}
	defer ee.Close(ctx)
	we := ee.(*wazeroExtEngine)
	err = we.selectModule(testPkg)
	if err != nil {
		panic(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		we.recover(context.Background())
	}
}

/*
goos: linux
goarch: amd64
cpu: 12th Gen Intel(R) Core(TM) i7-12700
Benchmark_Recover/2Mib-1%-20         	    6808	    168937 ns/op	 2221915 B/op	      61 allocs/op
Benchmark_Recover/2Mib-50%-20        	    6200	    178177 ns/op	 2221917 B/op	      61 allocs/op
Benchmark_Recover/2Mib-100%-20       	    6007	    191219 ns/op	 2233567 B/op	      63 allocs/op
Benchmark_Recover/8Mib-100%-20       	    1484	    759637 ns/op	 8525009 B/op	      63 allocs/op
Benchmark_Recover/100Mib-70%-20      	     117	   9026536 ns/op	100078791 B/op	      63 allocs/op
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
