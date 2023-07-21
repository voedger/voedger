# `objcache` Benchmark results

## Results

### 1. hashicorp

#### Sequenced Put/Get

```text
Running tool: D:\Go\bin\go.exe test -benchmem -run=^$ -bench ^Benchmark_CachePLogEvents$ github.com/voedger/voedger/pkg/objcache

goos: windows
goarch: amd64
pkg: github.com/voedger/voedger/pkg/objcache
cpu: Intel(R) Core(TM) i5-3570 CPU @ 3.40GHz
Benchmark_CachePLogEvents/Put-100-4         	   44443	     25853 ns/op	   15209 B/op	     116 allocs/op
Benchmark_CachePLogEvents/Get-100-4         	  152514	      8462 ns/op	       0 B/op	       0 allocs/op
	— Put:	       258 ns/op; Get:	        84 ns/op
Benchmark_CachePLogEvents/Put-1000-4        	    3672	    300430 ns/op	  187076 B/op	    1048 allocs/op
Benchmark_CachePLogEvents/Get-1000-4        	   14787	     81860 ns/op	       0 B/op	       0 allocs/op
	— Put:	       300 ns/op; Get:	        81 ns/op
Benchmark_CachePLogEvents/Put-10000-4       	     444	   2654976 ns/op	 1615434 B/op	   10289 allocs/op
Benchmark_CachePLogEvents/Get-10000-4       	    1405	    869607 ns/op	       0 B/op	       0 allocs/op
	— Put:	       265 ns/op; Get:	        86 ns/op
Benchmark_CachePLogEvents/Put-100000-4      	      37	  34030905 ns/op	14488878 B/op	  104008 allocs/op
Benchmark_CachePLogEvents/Get-100000-4      	      70	  20262880 ns/op	       0 B/op	       0 allocs/op
	— Put:	       340 ns/op; Get:	       202 ns/op
Benchmark_CachePLogEvents/Put-1000000-4     	       3	 544169400 ns/op	187830594 B/op	 1038222 allocs/op
Benchmark_CachePLogEvents/Get-1000000-4     	       6	 204052883 ns/op	       1 B/op	       0 allocs/op
	— Put:	       544 ns/op; Get:	       204 ns/op
PASS
ok  	github.com/voedger/voedger/pkg/objcache	24.395s
```

#### Parallel Put/Get

```text
Running tool: D:\Go\bin\go.exe test -benchmem -run=^$ -bench ^Benchmark_CachePLogEventsParallel$ github.com/voedger/voedger/pkg/objcache

goos: windows
goarch: amd64
pkg: github.com/voedger/voedger/pkg/objcache
cpu: Intel(R) Core(TM) i5-3570 CPU @ 3.40GHz
Benchmark_CachePLogEventsParallel/Put-100-10-4         	    2532	    474869 ns/op	  187988 B/op	    1069 allocs/op
Benchmark_CachePLogEventsParallel/Get-100-10-4         	    4525	    233667 ns/op	     898 B/op	      21 allocs/op
	— Put:	       474 ns/op; Get:	       233 ns/op
Benchmark_CachePLogEventsParallel/Put-500-50-4         	     117	   9974730 ns/op	 3627680 B/op	   26059 allocs/op
Benchmark_CachePLogEventsParallel/Get-500-50-4         	     142	   8093344 ns/op	    4451 B/op	     101 allocs/op
	— Put:	       398 ns/op; Get:	       323 ns/op
Benchmark_CachePLogEventsParallel/Put-1000-100-4       	      21	  47916943 ns/op	14497896 B/op	  104211 allocs/op
Benchmark_CachePLogEventsParallel/Get-1000-100-4       	      39	  31381572 ns/op	    9108 B/op	     203 allocs/op
	— Put:	       479 ns/op; Get:	       313 ns/op
PASS
ok  	github.com/voedger/voedger/pkg/objcache	9.115s
```