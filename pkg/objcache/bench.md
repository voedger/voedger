# `objcache` Benchmark results

## Results

```text
Running tool: D:\Go\bin\go.exe test -benchmem -run=^$ -bench ^BenchmarkCacheGeneralHashicorp$ github.com/voedger/voedger/pkg/objcache

goos: windows
goarch: amd64
pkg: github.com/voedger/voedger/pkg/objcache
cpu: Intel(R) Core(TM) i5-3570 CPU @ 3.40GHz
BenchmarkCacheGeneralHashicorp/1._Small_cache_100_events/1.1._Sequenced/Hashicorp-Seq-Put-100-4         	   46047	     26144 ns/op	   15211 B/op	     116 allocs/op
BenchmarkCacheGeneralHashicorp/1._Small_cache_100_events/1.1._Sequenced/Hashicorp-Seq-Get-100-4         	  164077	      7864 ns/op	       0 B/op	       0 allocs/op
	— Hashicorp:	 (Sequenced)	 Put:	       261 ns/op; Get:	        78 ns/op
BenchmarkCacheGeneralHashicorp/1._Small_cache_100_events/1.2._Parallel_(2×50)/Hashicorp-Par-Put-50-2-4  	   29967	     43315 ns/op	   15403 B/op	     121 allocs/op
BenchmarkCacheGeneralHashicorp/1._Small_cache_100_events/1.2._Parallel_(2×50)/Hashicorp-Par-Get-50-2-4  	   82412	     15037 ns/op	     192 B/op	       5 allocs/op
	— Hashicorp:	 (Parallel-2)	 Put:	       433 ns/op; Get:	       150 ns/op
BenchmarkCacheGeneralHashicorp/2._Middle_cache_1’000_events/2.1._Sequenced/Hashicorp-Seq-Put-1000-4     	    4137	    309117 ns/op	  187057 B/op	    1048 allocs/op
BenchmarkCacheGeneralHashicorp/2._Middle_cache_1’000_events/2.1._Sequenced/Hashicorp-Seq-Get-1000-4     	   15458	     78974 ns/op	       0 B/op	       0 allocs/op
	— Hashicorp:	 (Sequenced)	 Put:	       309 ns/op; Get:	        78 ns/op
BenchmarkCacheGeneralHashicorp/2._Middle_cache_1’000_events/2.2._Parallel_(10×100)/Hashicorp-Par-Put-100-10-4            	    2440	    468361 ns/op	  188003 B/op	    1069 allocs/op
BenchmarkCacheGeneralHashicorp/2._Middle_cache_1’000_events/2.2._Parallel_(10×100)/Hashicorp-Par-Get-100-10-4            	    6000	    253779 ns/op	     899 B/op	      21 allocs/op
	— Hashicorp:	 (Parallel-10)	 Put:	       468 ns/op; Get:	       253 ns/op
BenchmarkCacheGeneralHashicorp/3._Big_cache_10’000_events/3.1._Sequenced/Hashicorp-Seq-Put-10000-4                       	     421	   2700528 ns/op	 1615487 B/op	   10289 allocs/op
BenchmarkCacheGeneralHashicorp/3._Big_cache_10’000_events/3.1._Sequenced/Hashicorp-Seq-Get-10000-4                       	    1432	    881061 ns/op	       0 B/op	       0 allocs/op
	— Hashicorp:	 (Sequenced)	 Put:	       270 ns/op; Get:	        88 ns/op
BenchmarkCacheGeneralHashicorp/3._Big_cache_10’000_events/3.2._Parallel_(20×500)/Hashicorp-Par-Put-500-20-4              	     280	   4227757 ns/op	 1616954 B/op	   10329 allocs/op
BenchmarkCacheGeneralHashicorp/3._Big_cache_10’000_events/3.2._Parallel_(20×500)/Hashicorp-Par-Get-500-20-4              	     399	   2966633 ns/op	    1793 B/op	      41 allocs/op
	— Hashicorp:	 (Parallel-20)	 Put:	       422 ns/op; Get:	       296 ns/op
BenchmarkCacheGeneralHashicorp/4._Large_cache_100’000_events/3.1._Sequenced/Hashicorp-Seq-Put-100000-4                   	      36	  30243231 ns/op	14487036 B/op	  103999 allocs/op
BenchmarkCacheGeneralHashicorp/4._Large_cache_100’000_events/3.1._Sequenced/Hashicorp-Seq-Get-100000-4                   	      85	  16338988 ns/op	       0 B/op	       0 allocs/op
	— Hashicorp:	 (Sequenced)	 Put:	       302 ns/op; Get:	       163 ns/op
BenchmarkCacheGeneralHashicorp/4._Large_cache_100’000_events/3.2._Parallel_(100×1000)/Hashicorp-Par-Put-1000-100-4       	      25	  45697420 ns/op	14495254 B/op	  104199 allocs/op
BenchmarkCacheGeneralHashicorp/4._Large_cache_100’000_events/3.2._Parallel_(100×1000)/Hashicorp-Par-Get-1000-100-4       	      34	  34895209 ns/op	    9036 B/op	     203 allocs/op
	— Hashicorp:	 (Parallel-100)	 Put:	       456 ns/op; Get:	       348 ns/op
PASS
ok  	github.com/voedger/voedger/pkg/objcache	24.436s
```

```txt
Running tool: D:\Go\bin\go.exe test -benchmem -run=^$ -bench ^BenchmarkCacheGeneralTheine$ github.com/voedger/voedger/pkg/objcache

goos: windows
goarch: amd64
pkg: github.com/voedger/voedger/pkg/objcache
cpu: Intel(R) Core(TM) i5-3570 CPU @ 3.40GHz
BenchmarkCacheGeneralTheine/1._Small_cache_100_events/1.1._Sequenced/Theine-Seq-Put-100-4         	    6082	    202105 ns/op	   49285 B/op	     462 allocs/op
BenchmarkCacheGeneralTheine/1._Small_cache_100_events/1.1._Sequenced/Theine-Seq-Get-100-4         	   52164	     22858 ns/op	       0 B/op	       0 allocs/op
	— Theine:	 (Sequenced)	 Put:	      2021 ns/op; Get:	       228 ns/op
BenchmarkCacheGeneralTheine/1._Small_cache_100_events/1.2._Parallel_(2×50)/Theine-Par-Put-50-2-4  	    8880	    183616 ns/op	   49032 B/op	     466 allocs/op
BenchmarkCacheGeneralTheine/1._Small_cache_100_events/1.2._Parallel_(2×50)/Theine-Par-Get-50-2-4  	   45476	     25117 ns/op	     192 B/op	       5 allocs/op
	— Theine:	 (Parallel-2)	 Put:	      1836 ns/op; Get:	       251 ns/op
BenchmarkCacheGeneralTheine/2._Middle_cache_1’000_events/2.1._Sequenced/Theine-Seq-Put-1000-4     	    1072	   1114225 ns/op	  264523 B/op	    1478 allocs/op
BenchmarkCacheGeneralTheine/2._Middle_cache_1’000_events/2.1._Sequenced/Theine-Seq-Get-1000-4     	    5454	    233584 ns/op	       1 B/op	       0 allocs/op
	— Theine:	 (Sequenced)	 Put:	      1114 ns/op; Get:	       233 ns/op
BenchmarkCacheGeneralTheine/2._Middle_cache_1’000_events/2.2._Parallel_(10×100)/Theine-Par-Put-100-10-4            	    2498	    588270 ns/op	  269723 B/op	    1501 allocs/op
BenchmarkCacheGeneralTheine/2._Middle_cache_1’000_events/2.2._Parallel_(10×100)/Theine-Par-Get-100-10-4            	   10000	    114334 ns/op	     912 B/op	      21 allocs/op
	— Theine:	 (Parallel-10)	 Put:	       588 ns/op; Get:	       114 ns/op
BenchmarkCacheGeneralTheine/3._Big_cache_10’000_events/3.1._Sequenced/Theine-Seq-Put-10000-4                       	     169	   7247201 ns/op	 2242579 B/op	   10725 allocs/op
BenchmarkCacheGeneralTheine/3._Big_cache_10’000_events/3.1._Sequenced/Theine-Seq-Get-10000-4                       	     477	   2469586 ns/op	      14 B/op	       0 allocs/op
	— Theine:	 (Sequenced)	 Put:	       724 ns/op; Get:	       246 ns/op
BenchmarkCacheGeneralTheine/3._Big_cache_10’000_events/3.2._Parallel_(20×500)/Theine-Par-Put-500-20-4              	     303	  11543313 ns/op	 2243851 B/op	   10764 allocs/op
BenchmarkCacheGeneralTheine/3._Big_cache_10’000_events/3.2._Parallel_(20×500)/Theine-Par-Get-500-20-4              	    1471	    823591 ns/op	    2446 B/op	      42 allocs/op
	— Theine:	 (Parallel-20)	 Put:	      1154 ns/op; Get:	        82 ns/op
BenchmarkCacheGeneralTheine/4._Large_cache_100’000_events/3.1._Sequenced/Theine-Seq-Put-100000-4                   	      16	  83282762 ns/op	19869874 B/op	  104154 allocs/op
BenchmarkCacheGeneralTheine/4._Large_cache_100’000_events/3.1._Sequenced/Theine-Seq-Get-100000-4                   	      34	  38475574 ns/op	     202 B/op	       0 allocs/op
	— Theine:	 (Sequenced)	 Put:	       832 ns/op; Get:	       384 ns/op
BenchmarkCacheGeneralTheine/4._Large_cache_100’000_events/3.2._Parallel_(100×1000)/Theine-Par-Put-1000-100-4       	      33	 166193382 ns/op	19883453 B/op	  104377 allocs/op
BenchmarkCacheGeneralTheine/4._Large_cache_100’000_events/3.2._Parallel_(100×1000)/Theine-Par-Get-1000-100-4       	     112	  10315779 ns/op	   22890 B/op	     201 allocs/op
	— Theine:	 (Parallel-100)	 Put:	      1661 ns/op; Get:	       103 ns/op
PASS
ok  	github.com/voedger/voedger/pkg/objcache	72.576s
```

```text
Running tool: D:\Go\bin\go.exe test -benchmem -run=^$ -bench ^BenchmarkCacheGeneralFloatdrop$ github.com/voedger/voedger/pkg/objcache

goos: windows
goarch: amd64
pkg: github.com/voedger/voedger/pkg/objcache
cpu: Intel(R) Core(TM) i5-3570 CPU @ 3.40GHz
BenchmarkCacheGeneralFloatdrop/1._Small_cache_100_events/1.1._Sequenced/Floatdrop-Seq-Put-100-4         	   39700	     33154 ns/op	   11524 B/op	     306 allocs/op
BenchmarkCacheGeneralFloatdrop/1._Small_cache_100_events/1.1._Sequenced/Floatdrop-Seq-Get-100-4         	  172318	      6901 ns/op	       0 B/op	       0 allocs/op
	— Floatdrop:	 (Sequenced)	 Put:	       331 ns/op; Get:	        69 ns/op
BenchmarkCacheGeneralFloatdrop/1._Small_cache_100_events/1.2._Parallel_(2×50)/Floatdrop-Par-Put-50-2-4  	   27726	     44205 ns/op	   11717 B/op	     311 allocs/op
BenchmarkCacheGeneralFloatdrop/1._Small_cache_100_events/1.2._Parallel_(2×50)/Floatdrop-Par-Get-50-2-4  	   83119	     14254 ns/op	     192 B/op	       5 allocs/op
	— Floatdrop:	 (Parallel-2)	 Put:	       442 ns/op; Get:	       142 ns/op
BenchmarkCacheGeneralFloatdrop/2._Middle_cache_1’000_events/2.1._Sequenced/Floatdrop-Seq-Put-1000-4     	    3994	    298928 ns/op	  129512 B/op	    3006 allocs/op
BenchmarkCacheGeneralFloatdrop/2._Middle_cache_1’000_events/2.1._Sequenced/Floatdrop-Seq-Get-1000-4     	   16044	     74115 ns/op	       0 B/op	       0 allocs/op
	— Floatdrop:	 (Sequenced)	 Put:	       298 ns/op; Get:	        74 ns/op
BenchmarkCacheGeneralFloatdrop/2._Middle_cache_1’000_events/2.2._Parallel_(10×100)/Floatdrop-Par-Put-100-10-4            	    2408	    493750 ns/op	  130419 B/op	    3027 allocs/op
BenchmarkCacheGeneralFloatdrop/2._Middle_cache_1’000_events/2.2._Parallel_(10×100)/Floatdrop-Par-Get-100-10-4            	    5448	    229614 ns/op	     900 B/op	      21 allocs/op
	— Floatdrop:	 (Parallel-10)	 Put:	       493 ns/op; Get:	       229 ns/op
BenchmarkCacheGeneralFloatdrop/3._Big_cache_10’000_events/3.1._Sequenced/Floatdrop-Seq-Put-10000-4                       	     406	   2867865 ns/op	 1178924 B/op	   30006 allocs/op
BenchmarkCacheGeneralFloatdrop/3._Big_cache_10’000_events/3.1._Sequenced/Floatdrop-Seq-Get-10000-4                       	    1578	    807278 ns/op	       0 B/op	       0 allocs/op
	— Floatdrop:	 (Sequenced)	 Put:	       286 ns/op; Get:	        80 ns/op
BenchmarkCacheGeneralFloatdrop/3._Big_cache_10’000_events/3.2._Parallel_(20×500)/Floatdrop-Par-Put-500-20-4              	     247	   4798548 ns/op	 1180718 B/op	   30047 allocs/op
BenchmarkCacheGeneralFloatdrop/3._Big_cache_10’000_events/3.2._Parallel_(20×500)/Floatdrop-Par-Get-500-20-4              	     438	   2622757 ns/op	    1777 B/op	      41 allocs/op
	— Floatdrop:	 (Parallel-20)	 Put:	       479 ns/op; Get:	       262 ns/op
BenchmarkCacheGeneralFloatdrop/4._Large_cache_100’000_events/3.1._Sequenced/Floatdrop-Seq-Put-100000-4                   	      31	  38576661 ns/op	11165178 B/op	  301660 allocs/op
BenchmarkCacheGeneralFloatdrop/4._Large_cache_100’000_events/3.1._Sequenced/Floatdrop-Seq-Get-100000-4                   	      63	  16773417 ns/op	       0 B/op	       0 allocs/op
	— Floatdrop:	 (Sequenced)	 Put:	       385 ns/op; Get:	       167 ns/op
BenchmarkCacheGeneralFloatdrop/4._Large_cache_100’000_events/3.2._Parallel_(100×1000)/Floatdrop-Par-Put-1000-100-4       	      21	  51614019 ns/op	11176176 B/op	  301869 allocs/op
BenchmarkCacheGeneralFloatdrop/4._Large_cache_100’000_events/3.2._Parallel_(100×1000)/Floatdrop-Par-Get-1000-100-4       	      39	  31268567 ns/op	    9072 B/op	     203 allocs/op
	— Floatdrop:	 (Parallel-100)	 Put:	       516 ns/op; Get:	       312 ns/op
PASS
ok  	github.com/voedger/voedger/pkg/objcache	23.871s
```

```txt
Running tool: D:\Go\bin\go.exe test -benchmem -run=^$ -bench ^BenchmarkCacheGeneralImcache$ github.com/voedger/voedger/pkg/objcache

goos: windows
goarch: amd64
pkg: github.com/voedger/voedger/pkg/objcache
cpu: Intel(R) Core(TM) i5-3570 CPU @ 3.40GHz
BenchmarkCacheGeneralImcache/1._Small_cache_100_events/1.1._Sequenced/Imcache-Seq-Put-100-4         	   34545	     33359 ns/op	   26230 B/op	     218 allocs/op
BenchmarkCacheGeneralImcache/1._Small_cache_100_events/1.1._Sequenced/Imcache-Seq-Get-100-4         	  160992	      7471 ns/op	       0 B/op	       0 allocs/op
	— Imcache:	 (Sequenced)	 Put:	       333 ns/op; Get:	        74 ns/op
BenchmarkCacheGeneralImcache/1._Small_cache_100_events/1.2._Parallel_(2×50)/Imcache-Par-Put-50-2-4  	   22195	     54166 ns/op	   26413 B/op	     223 allocs/op
BenchmarkCacheGeneralImcache/1._Small_cache_100_events/1.2._Parallel_(2×50)/Imcache-Par-Get-50-2-4  	   83616	     14259 ns/op	     192 B/op	       5 allocs/op
	— Imcache:	 (Parallel-2)	 Put:	       541 ns/op; Get:	       142 ns/op
BenchmarkCacheGeneralImcache/2._Middle_cache_1’000_events/2.1._Sequenced/Imcache-Seq-Put-1000-4     	    2998	    436877 ns/op	  355919 B/op	    2047 allocs/op
BenchmarkCacheGeneralImcache/2._Middle_cache_1’000_events/2.1._Sequenced/Imcache-Seq-Get-1000-4     	   14568	     86515 ns/op	       0 B/op	       0 allocs/op
	— Imcache:	 (Sequenced)	 Put:	       436 ns/op; Get:	        86 ns/op
BenchmarkCacheGeneralImcache/2._Middle_cache_1’000_events/2.2._Parallel_(10×100)/Imcache-Par-Put-100-10-4            	    1904	    611287 ns/op	  356910 B/op	    2068 allocs/op
BenchmarkCacheGeneralImcache/2._Middle_cache_1’000_events/2.2._Parallel_(10×100)/Imcache-Par-Get-100-10-4            	    4894	    241071 ns/op	     898 B/op	      21 allocs/op
	— Imcache:	 (Parallel-10)	 Put:	       611 ns/op; Get:	       241 ns/op
BenchmarkCacheGeneralImcache/3._Big_cache_10’000_events/3.1._Sequenced/Imcache-Seq-Put-10000-4                       	     315	   3834050 ns/op	 2969493 B/op	   20291 allocs/op
BenchmarkCacheGeneralImcache/3._Big_cache_10’000_events/3.1._Sequenced/Imcache-Seq-Get-10000-4                       	    1294	    899718 ns/op	       0 B/op	       0 allocs/op
	— Imcache:	 (Sequenced)	 Put:	       383 ns/op; Get:	        89 ns/op
BenchmarkCacheGeneralImcache/3._Big_cache_10’000_events/3.2._Parallel_(20×500)/Imcache-Par-Put-500-20-4              	     220	   5298648 ns/op	 2971499 B/op	   20333 allocs/op
BenchmarkCacheGeneralImcache/3._Big_cache_10’000_events/3.2._Parallel_(20×500)/Imcache-Par-Get-500-20-4              	     390	   3070017 ns/op	    1778 B/op	      41 allocs/op
	— Imcache:	 (Parallel-20)	 Put:	       529 ns/op; Get:	       307 ns/op
BenchmarkCacheGeneralImcache/4._Large_cache_100’000_events/3.1._Sequenced/Imcache-Seq-Put-100000-4                   	      24	  41888617 ns/op	26108452 B/op	  204043 allocs/op
BenchmarkCacheGeneralImcache/4._Large_cache_100’000_events/3.1._Sequenced/Imcache-Seq-Get-100000-4                   	      62	  16304469 ns/op	       0 B/op	       0 allocs/op
	— Imcache:	 (Sequenced)	 Put:	       418 ns/op; Get:	       163 ns/op
BenchmarkCacheGeneralImcache/4._Large_cache_100’000_events/3.2._Parallel_(100×1000)/Imcache-Par-Put-1000-100-4       	      19	  60722732 ns/op	26119742 B/op	  204257 allocs/op
BenchmarkCacheGeneralImcache/4._Large_cache_100’000_events/3.2._Parallel_(100×1000)/Imcache-Par-Get-1000-100-4       	      37	  32419335 ns/op	    9265 B/op	     204 allocs/op
	— Imcache:	 (Parallel-100)	 Put:	       607 ns/op; Get:	       324 ns/op
PASS
ok  	github.com/voedger/voedger/pkg/objcache	22.992s
```

```text
Running tool: D:\Go\bin\go.exe test -benchmem -run=^$ -bench ^BenchmarkCacheParallelismHashicorp$ github.com/voedger/voedger/pkg/objcache

goos: windows
goarch: amd64
pkg: github.com/voedger/voedger/pkg/objcache
cpu: Intel(R) Core(TM) i5-3570 CPU @ 3.40GHz
BenchmarkCacheParallelismHashicorp/Hashicorp-Par-Put-1000-1-4         	    3678	    307386 ns/op	  187200 B/op	    1051 allocs/op
BenchmarkCacheParallelismHashicorp/Hashicorp-Par-Get-1000-1-4         	   14037	     84857 ns/op	     104 B/op	       3 allocs/op
	— Hashicorp:	 (Parallel-1)	 Put:	       307 ns/op; Get:	        84 ns/op
BenchmarkCacheParallelismHashicorp/Hashicorp-Par-Put-1000-11-4        	     255	   4744655 ns/op	 1687974 B/op	   11348 allocs/op
BenchmarkCacheParallelismHashicorp/Hashicorp-Par-Get-1000-11-4        	     369	   3111294 ns/op	    1001 B/op	      23 allocs/op
	— Hashicorp:	 (Parallel-11)	 Put:	       431 ns/op; Get:	       282 ns/op
BenchmarkCacheParallelismHashicorp/Hashicorp-Par-Put-1000-21-4        	     128	   8986643 ns/op	 3295278 B/op	   21644 allocs/op
BenchmarkCacheParallelismHashicorp/Hashicorp-Par-Get-1000-21-4        	     171	   6578802 ns/op	    1920 B/op	      43 allocs/op
	— Hashicorp:	 (Parallel-21)	 Put:	       427 ns/op; Get:	       313 ns/op
BenchmarkCacheParallelismHashicorp/Hashicorp-Par-Put-1000-31-4        	      69	  14872819 ns/op	 5858886 B/op	   32205 allocs/op
BenchmarkCacheParallelismHashicorp/Hashicorp-Par-Get-1000-31-4        	     100	  11026333 ns/op	    2819 B/op	      63 allocs/op
	— Hashicorp:	 (Parallel-31)	 Put:	       479 ns/op; Get:	       355 ns/op
BenchmarkCacheParallelismHashicorp/Hashicorp-Par-Put-1000-41-4        	      56	  18638180 ns/op	 6510610 B/op	   42278 allocs/op
BenchmarkCacheParallelismHashicorp/Hashicorp-Par-Get-1000-41-4        	      85	  13768519 ns/op	    3697 B/op	      83 allocs/op
	— Hashicorp:	 (Parallel-41)	 Put:	       454 ns/op; Get:	       335 ns/op
BenchmarkCacheParallelismHashicorp/Hashicorp-Par-Put-1000-51-4        	      49	  22793047 ns/op	 7335202 B/op	   53181 allocs/op
BenchmarkCacheParallelismHashicorp/Hashicorp-Par-Get-1000-51-4        	      64	  18501627 ns/op	    4650 B/op	     103 allocs/op
	— Hashicorp:	 (Parallel-51)	 Put:	       446 ns/op; Get:	       362 ns/op
BenchmarkCacheParallelismHashicorp/Hashicorp-Par-Put-1000-61-4        	      39	  30967418 ns/op	11651697 B/op	   63467 allocs/op
BenchmarkCacheParallelismHashicorp/Hashicorp-Par-Get-1000-61-4        	      54	  22318541 ns/op	    5462 B/op	     123 allocs/op
	— Hashicorp:	 (Parallel-61)	 Put:	       507 ns/op; Get:	       365 ns/op
BenchmarkCacheParallelismHashicorp/Hashicorp-Par-Put-1000-71-4        	      28	  37700764 ns/op	12292208 B/op	   73484 allocs/op
BenchmarkCacheParallelismHashicorp/Hashicorp-Par-Get-1000-71-4        	      44	  26554148 ns/op	    6586 B/op	     144 allocs/op
	— Hashicorp:	 (Parallel-71)	 Put:	       530 ns/op; Get:	       374 ns/op
BenchmarkCacheParallelismHashicorp/Hashicorp-Par-Put-1000-81-4        	      32	  39399722 ns/op	12940139 B/op	   83540 allocs/op
BenchmarkCacheParallelismHashicorp/Hashicorp-Par-Get-1000-81-4        	      42	  28889957 ns/op	    7348 B/op	     164 allocs/op
	— Hashicorp:	 (Parallel-81)	 Put:	       486 ns/op; Get:	       356 ns/op
BenchmarkCacheParallelismHashicorp/Hashicorp-Par-Put-1000-91-4        	      27	  42693893 ns/op	13734482 B/op	   94296 allocs/op
BenchmarkCacheParallelismHashicorp/Hashicorp-Par-Get-1000-91-4        	      38	  31784224 ns/op	    8269 B/op	     184 allocs/op
	— Hashicorp:	 (Parallel-91)	 Put:	       469 ns/op; Get:	       349 ns/op
BenchmarkCacheParallelismHashicorp/Hashicorp-Par-Put-1000-101-4       	      25	  48191364 ns/op	14582262 B/op	  105313 allocs/op
BenchmarkCacheParallelismHashicorp/Hashicorp-Par-Get-1000-101-4       	      34	  35979326 ns/op	    9172 B/op	     205 allocs/op
	— Hashicorp:	 (Parallel-101)	 Put:	       477 ns/op; Get:	       356 ns/op
PASS
ok  	github.com/voedger/voedger/pkg/objcache	41.397s
```

```text
Running tool: D:\Go\bin\go.exe test -benchmem -run=^$ -bench ^BenchmarkCacheParallelismTheine$ github.com/voedger/voedger/pkg/objcache

goos: windows
goarch: amd64
pkg: github.com/voedger/voedger/pkg/objcache
cpu: Intel(R) Core(TM) i5-3570 CPU @ 3.40GHz
BenchmarkCacheParallelismTheine/Theine-Par-Put-1000-1-4         	    1068	   1140664 ns/op	  264562 B/op	    1480 allocs/op
BenchmarkCacheParallelismTheine/Theine-Par-Get-1000-1-4         	    5216	    247089 ns/op	     105 B/op	       3 allocs/op
	— Theine:	 (Parallel-1)	 Put:	      1140 ns/op; Get:	       247 ns/op
BenchmarkCacheParallelismTheine/Theine-Par-Put-1000-11-4        	     188	   6258138 ns/op	 2349037 B/op	   11793 allocs/op
BenchmarkCacheParallelismTheine/Theine-Par-Get-1000-11-4        	     805	   1296985 ns/op	    2133 B/op	      23 allocs/op
	— Theine:	 (Parallel-11)	 Put:	       568 ns/op; Get:	       117 ns/op
BenchmarkCacheParallelismTheine/Theine-Par-Put-1000-21-4        	     136	   9582185 ns/op	 4545669 B/op	   22087 allocs/op
BenchmarkCacheParallelismTheine/Theine-Par-Get-1000-21-4        	     465	   2176638 ns/op	    3874 B/op	      44 allocs/op
	— Theine:	 (Parallel-21)	 Put:	       456 ns/op; Get:	       103 ns/op
BenchmarkCacheParallelismTheine/Theine-Par-Put-1000-31-4        	      98	  14691877 ns/op	 7455996 B/op	   32657 allocs/op
BenchmarkCacheParallelismTheine/Theine-Par-Get-1000-31-4        	     343	   2965776 ns/op	    6483 B/op	      74 allocs/op
	— Theine:	 (Parallel-31)	 Put:	       473 ns/op; Get:	        95 ns/op
BenchmarkCacheParallelismTheine/Theine-Par-Put-1000-41-4        	      79	  16241305 ns/op	 8945752 B/op	   42689 allocs/op
BenchmarkCacheParallelismTheine/Theine-Par-Get-1000-41-4        	     295	   3820910 ns/op	    6309 B/op	      83 allocs/op
	— Theine:	 (Parallel-41)	 Put:	       396 ns/op; Get:	        93 ns/op
BenchmarkCacheParallelismTheine/Theine-Par-Put-1000-51-4        	      66	  18718039 ns/op	10078026 B/op	   53524 allocs/op
BenchmarkCacheParallelismTheine/Theine-Par-Get-1000-51-4        	     267	   4772201 ns/op	   11408 B/op	     103 allocs/op
	— Theine:	 (Parallel-51)	 Put:	       367 ns/op; Get:	        93 ns/op
BenchmarkCacheParallelismTheine/Theine-Par-Put-1000-61-4        	      46	  65454067 ns/op	14774675 B/op	   63834 allocs/op
BenchmarkCacheParallelismTheine/Theine-Par-Get-1000-61-4        	     200	   6260724 ns/op	   14037 B/op	     155 allocs/op
	— Theine:	 (Parallel-61)	 Put:	      1071 ns/op; Get:	       102 ns/op
BenchmarkCacheParallelismTheine/Theine-Par-Put-1000-71-4        	      46	  46479300 ns/op	16787471 B/op	   73858 allocs/op
BenchmarkCacheParallelismTheine/Theine-Par-Get-1000-71-4        	     168	   7747642 ns/op	   14125 B/op	     143 allocs/op
	— Theine:	 (Parallel-71)	 Put:	       654 ns/op; Get:	       109 ns/op
BenchmarkCacheParallelismTheine/Theine-Par-Put-1000-81-4        	      36	  95338597 ns/op	17749724 B/op	   83877 allocs/op
BenchmarkCacheParallelismTheine/Theine-Par-Get-1000-81-4        	     100	  11630175 ns/op	   18157 B/op	     263 allocs/op
	— Theine:	 (Parallel-81)	 Put:	      1176 ns/op; Get:	       143 ns/op
BenchmarkCacheParallelismTheine/Theine-Par-Put-1000-91-4        	      36	  60234867 ns/op	18829315 B/op	   94451 allocs/op
BenchmarkCacheParallelismTheine/Theine-Par-Get-1000-91-4        	     139	   9449111 ns/op	   19358 B/op	     183 allocs/op
	— Theine:	 (Parallel-91)	 Put:	       661 ns/op; Get:	       103 ns/op
BenchmarkCacheParallelismTheine/Theine-Par-Put-1000-101-4       	      26	  71796077 ns/op	20000371 B/op	  105481 allocs/op
BenchmarkCacheParallelismTheine/Theine-Par-Get-1000-101-4       	     105	  11728340 ns/op	   43198 B/op	     485 allocs/op
	— Theine:	 (Parallel-101)	 Put:	       710 ns/op; Get:	       116 ns/op
PASS
ok  	github.com/voedger/voedger/pkg/objcache	186.773s
```

```text
Running tool: D:\Go\bin\go.exe test -benchmem -run=^$ -bench ^BenchmarkCacheParallelismFloatdrop$ github.com/voedger/voedger/pkg/objcache

goos: windows
goarch: amd64
pkg: github.com/voedger/voedger/pkg/objcache
cpu: Intel(R) Core(TM) i5-3570 CPU @ 3.40GHz
BenchmarkCacheParallelismFloatdrop/Floatdrop-Par-Put-1000-1-4         	    3596	    317497 ns/op	  129619 B/op	    3009 allocs/op
BenchmarkCacheParallelismFloatdrop/Floatdrop-Par-Get-1000-1-4         	   14637	     77474 ns/op	     104 B/op	       3 allocs/op
	— Floatdrop:	 (Parallel-1)	 Put:	       317 ns/op; Get:	        77 ns/op
BenchmarkCacheParallelismFloatdrop/Floatdrop-Par-Put-1000-11-4        	     230	   5109857 ns/op	 1259711 B/op	   33066 allocs/op
BenchmarkCacheParallelismFloatdrop/Floatdrop-Par-Get-1000-11-4        	     460	   2938049 ns/op	    1001 B/op	      23 allocs/op
	— Floatdrop:	 (Parallel-11)	 Put:	       464 ns/op; Get:	       267 ns/op
BenchmarkCacheParallelismFloatdrop/Floatdrop-Par-Put-1000-21-4        	     100	  10492879 ns/op	 2431888 B/op	   63090 allocs/op
BenchmarkCacheParallelismFloatdrop/Floatdrop-Par-Get-1000-21-4        	     187	   6603913 ns/op	    1888 B/op	      43 allocs/op
	— Floatdrop:	 (Parallel-21)	 Put:	       499 ns/op; Get:	       314 ns/op
BenchmarkCacheParallelismFloatdrop/Floatdrop-Par-Put-1000-31-4        	      69	  15658159 ns/op	 4045412 B/op	   93069 allocs/op
BenchmarkCacheParallelismFloatdrop/Floatdrop-Par-Get-1000-31-4        	     100	  10816964 ns/op	    2853 B/op	      63 allocs/op
	— Floatdrop:	 (Parallel-31)	 Put:	       505 ns/op; Get:	       348 ns/op
BenchmarkCacheParallelismFloatdrop/Floatdrop-Par-Put-1000-41-4        	      54	  21054917 ns/op	 4775966 B/op	  123136 allocs/op
BenchmarkCacheParallelismFloatdrop/Floatdrop-Par-Get-1000-41-4        	      91	  13614533 ns/op	    3762 B/op	      83 allocs/op
	— Floatdrop:	 (Parallel-41)	 Put:	       513 ns/op; Get:	       332 ns/op
BenchmarkCacheParallelismFloatdrop/Floatdrop-Par-Put-1000-51-4        	      38	  26372550 ns/op	 5681983 B/op	  154047 allocs/op
BenchmarkCacheParallelismFloatdrop/Floatdrop-Par-Get-1000-51-4        	      66	  17641898 ns/op	    4635 B/op	     104 allocs/op
	— Floatdrop:	 (Parallel-51)	 Put:	       517 ns/op; Get:	       345 ns/op
BenchmarkCacheParallelismFloatdrop/Floatdrop-Par-Put-1000-61-4        	      34	  32146279 ns/op	 8018519 B/op	  183129 allocs/op
BenchmarkCacheParallelismFloatdrop/Floatdrop-Par-Get-1000-61-4        	      57	  21016800 ns/op	    5522 B/op	     123 allocs/op
	— Floatdrop:	 (Parallel-61)	 Put:	       526 ns/op; Get:	       344 ns/op
BenchmarkCacheParallelismFloatdrop/Floatdrop-Par-Put-1000-71-4        	      31	  39006774 ns/op	 8739776 B/op	  213154 allocs/op
BenchmarkCacheParallelismFloatdrop/Floatdrop-Par-Get-1000-71-4        	      48	  24938356 ns/op	    6477 B/op	     145 allocs/op
	— Floatdrop:	 (Parallel-71)	 Put:	       549 ns/op; Get:	       351 ns/op
BenchmarkCacheParallelismFloatdrop/Floatdrop-Par-Put-1000-81-4        	      26	  43614604 ns/op	 9467870 B/op	  243206 allocs/op
BenchmarkCacheParallelismFloatdrop/Floatdrop-Par-Get-1000-81-4        	      39	  28576772 ns/op	    7556 B/op	     165 allocs/op
	— Floatdrop:	 (Parallel-81)	 Put:	       538 ns/op; Get:	       352 ns/op
BenchmarkCacheParallelismFloatdrop/Floatdrop-Par-Put-1000-91-4        	      25	  49369016 ns/op	10346317 B/op	  273984 allocs/op
BenchmarkCacheParallelismFloatdrop/Floatdrop-Par-Get-1000-91-4        	      39	  32160426 ns/op	    8277 B/op	     185 allocs/op
	— Floatdrop:	 (Parallel-91)	 Put:	       542 ns/op; Get:	       353 ns/op
BenchmarkCacheParallelismFloatdrop/Floatdrop-Par-Put-1000-101-4       	      21	  53835871 ns/op	11270391 B/op	  304982 allocs/op
BenchmarkCacheParallelismFloatdrop/Floatdrop-Par-Get-1000-101-4       	      34	  33932918 ns/op	    9208 B/op	     206 allocs/op
	— Floatdrop:	 (Parallel-101)	 Put:	       533 ns/op; Get:	       335 ns/op
PASS
ok  	github.com/voedger/voedger/pkg/objcache	40.497s
```

```text
Running tool: D:\Go\bin\go.exe test -benchmem -run=^$ -bench ^BenchmarkCacheParallelismImcache$ github.com/voedger/voedger/pkg/objcache

goos: windows
goarch: amd64
pkg: github.com/voedger/voedger/pkg/objcache
cpu: Intel(R) Core(TM) i5-3570 CPU @ 3.40GHz
BenchmarkCacheParallelismImcache/Imcache-Par-Put-1000-1-4         	    2589	    414451 ns/op	  356091 B/op	    2050 allocs/op
BenchmarkCacheParallelismImcache/Imcache-Par-Get-1000-1-4         	   13496	     89748 ns/op	     104 B/op	       3 allocs/op
	— Imcache:	 (Parallel-1)	 Put:	       414 ns/op; Get:	        89 ns/op
BenchmarkCacheParallelismImcache/Imcache-Par-Put-1000-11-4        	     189	   5995678 ns/op	 3078352 B/op	   22372 allocs/op
BenchmarkCacheParallelismImcache/Imcache-Par-Get-1000-11-4        	     400	   3109524 ns/op	    1014 B/op	      23 allocs/op
	— Imcache:	 (Parallel-11)	 Put:	       545 ns/op; Get:	       282 ns/op
BenchmarkCacheParallelismImcache/Imcache-Par-Put-1000-21-4        	     103	  11164650 ns/op	 6036871 B/op	   42674 allocs/op
BenchmarkCacheParallelismImcache/Imcache-Par-Get-1000-21-4        	     181	   6507617 ns/op	    1895 B/op	      43 allocs/op
	— Imcache:	 (Parallel-21)	 Put:	       531 ns/op; Get:	       309 ns/op
BenchmarkCacheParallelismImcache/Imcache-Par-Put-1000-31-4        	      62	  18544590 ns/op	11138797 B/op	   63243 allocs/op
BenchmarkCacheParallelismImcache/Imcache-Par-Get-1000-31-4        	     120	  10006134 ns/op	    2752 B/op	      63 allocs/op
	— Imcache:	 (Parallel-31)	 Put:	       598 ns/op; Get:	       322 ns/op
BenchmarkCacheParallelismImcache/Imcache-Par-Put-1000-41-4        	      50	  23175732 ns/op	11963045 B/op	   83312 allocs/op
BenchmarkCacheParallelismImcache/Imcache-Par-Get-1000-41-4        	      91	  13341621 ns/op	    3691 B/op	      83 allocs/op
	— Imcache:	 (Parallel-41)	 Put:	       565 ns/op; Get:	       325 ns/op
BenchmarkCacheParallelismImcache/Imcache-Par-Put-1000-51-4        	      42	  28102752 ns/op	13193097 B/op	  104225 allocs/op
BenchmarkCacheParallelismImcache/Imcache-Par-Get-1000-51-4        	      74	  16645423 ns/op	    4504 B/op	     103 allocs/op
	— Imcache:	 (Parallel-51)	 Put:	       551 ns/op; Get:	       326 ns/op
BenchmarkCacheParallelismImcache/Imcache-Par-Put-1000-61-4        	      27	  40763907 ns/op	22191536 B/op	  124504 allocs/op
BenchmarkCacheParallelismImcache/Imcache-Par-Get-1000-61-4        	      54	  21011093 ns/op	    5494 B/op	     123 allocs/op
	— Imcache:	 (Parallel-61)	 Put:	       668 ns/op; Get:	       344 ns/op
BenchmarkCacheParallelismImcache/Imcache-Par-Put-1000-71-4        	      26	  45834869 ns/op	22995568 B/op	  144529 allocs/op
BenchmarkCacheParallelismImcache/Imcache-Par-Get-1000-71-4        	      49	  24391698 ns/op	    6620 B/op	     146 allocs/op
	— Imcache:	 (Parallel-71)	 Put:	       645 ns/op; Get:	       343 ns/op
BenchmarkCacheParallelismImcache/Imcache-Par-Put-1000-81-4        	      24	  50768296 ns/op	23807898 B/op	  164571 allocs/op
BenchmarkCacheParallelismImcache/Imcache-Par-Get-1000-81-4        	      43	  27538681 ns/op	    7437 B/op	     165 allocs/op
	— Imcache:	 (Parallel-81)	 Put:	       626 ns/op; Get:	       339 ns/op
BenchmarkCacheParallelismImcache/Imcache-Par-Put-1000-91-4        	      21	  55270424 ns/op	24973270 B/op	  185351 allocs/op
BenchmarkCacheParallelismImcache/Imcache-Par-Get-1000-91-4        	      38	  31118745 ns/op	    8315 B/op	     185 allocs/op
	— Imcache:	 (Parallel-91)	 Put:	       607 ns/op; Get:	       341 ns/op
BenchmarkCacheParallelismImcache/Imcache-Par-Put-1000-101-4       	      19	  59131089 ns/op	26237865 B/op	  206337 allocs/op
BenchmarkCacheParallelismImcache/Imcache-Par-Get-1000-101-4       	      34	  34163671 ns/op	    9146 B/op	     205 allocs/op
	— Imcache:	 (Parallel-101)	 Put:	       585 ns/op; Get:	       338 ns/op
PASS
ok  	github.com/voedger/voedger/pkg/objcache	47.037s
```
