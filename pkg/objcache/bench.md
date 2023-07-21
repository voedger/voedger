# `objcache` Benchmark results

## Results

```txt
Running tool: D:\Go\bin\go.exe test -benchmem -run=^$ -bench ^BenchmarkAll$ github.com/voedger/voedger/pkg/objcache

goos: windows
goarch: amd64
pkg: github.com/voedger/voedger/pkg/objcache
cpu: Intel(R) Core(TM) i5-3570 CPU @ 3.40GHz
BenchmarkAll/1._Small_cache_100_events/1.1._Sequenced/Hashicorp-Seq-Put-100-4         	   42867	     25835 ns/op	   15209 B/op	     116 allocs/op
BenchmarkAll/1._Small_cache_100_events/1.1._Sequenced/Hashicorp-Seq-Get-100-4         	  166681	      7397 ns/op	       0 B/op	       0 allocs/op
	— Hashicorp:	 (Sequenced)	 Put:	       258 ns/op; Get:	        73 ns/op
BenchmarkAll/1._Small_cache_100_events/1.1._Sequenced/Theine-Seq-Put-100-4            	    6850	    194799 ns/op	   48843 B/op	     461 allocs/op
BenchmarkAll/1._Small_cache_100_events/1.1._Sequenced/Theine-Seq-Get-100-4            	   45661	     24057 ns/op	       2 B/op	       0 allocs/op
	— Theine:	 (Sequenced)	 Put:	      1947 ns/op; Get:	       240 ns/op
BenchmarkAll/1._Small_cache_100_events/1.2._Parallel/Hashicorp-Par-Put-50-2-4         	   21315	     58972 ns/op	   15404 B/op	     121 allocs/op
BenchmarkAll/1._Small_cache_100_events/1.2._Parallel/Hashicorp-Par-Get-50-2-4         	   81544	     14254 ns/op	     192 B/op	       5 allocs/op
	— Hashicorp:	 (Parallel)	 Put:	       589 ns/op; Get:	       142 ns/op
BenchmarkAll/1._Small_cache_100_events/1.2._Parallel/Theine-Par-Put-50-2-4            	    8340	    146029 ns/op	   49858 B/op	     468 allocs/op
BenchmarkAll/1._Small_cache_100_events/1.2._Parallel/Theine-Par-Get-50-2-4            	   47247	     25557 ns/op	     192 B/op	       5 allocs/op
	— Theine:	 (Parallel)	 Put:	      1460 ns/op; Get:	       255 ns/op
BenchmarkAll/2._Middle_cache_1’000_events/2.1._Sequenced/Hashicorp-Seq-Put-1000-4     	    3750	    364665 ns/op	  187104 B/op	    1048 allocs/op
BenchmarkAll/2._Middle_cache_1’000_events/2.1._Sequenced/Hashicorp-Seq-Get-1000-4     	   12714	     81102 ns/op	       0 B/op	       0 allocs/op
	— Hashicorp:	 (Sequenced)	 Put:	       364 ns/op; Get:	        81 ns/op
BenchmarkAll/2._Middle_cache_1’000_events/2.1._Sequenced/Theine-Seq-Put-1000-4        	    1353	    804341 ns/op	  264441 B/op	    1478 allocs/op
BenchmarkAll/2._Middle_cache_1’000_events/2.1._Sequenced/Theine-Seq-Get-1000-4        	    5217	    236225 ns/op	       1 B/op	       0 allocs/op
	— Theine:	 (Sequenced)	 Put:	       804 ns/op; Get:	       236 ns/op
BenchmarkAll/2._Middle_cache_1’000_events/2.2._Parallel/Hashicorp-Par-Put-100-10-4    	    2757	    504467 ns/op	  188013 B/op	    1069 allocs/op
BenchmarkAll/2._Middle_cache_1’000_events/2.2._Parallel/Hashicorp-Par-Get-100-10-4    	    5262	    239191 ns/op	     897 B/op	      21 allocs/op
	— Hashicorp:	 (Parallel)	 Put:	       504 ns/op; Get:	       239 ns/op
BenchmarkAll/2._Middle_cache_1’000_events/2.2._Parallel/Theine-Par-Put-100-10-4       	    2473	    492463 ns/op	  273925 B/op	    1504 allocs/op
BenchmarkAll/2._Middle_cache_1’000_events/2.2._Parallel/Theine-Par-Get-100-10-4       	   10564	    118386 ns/op	     908 B/op	      21 allocs/op
	— Theine:	 (Parallel)	 Put:	       492 ns/op; Get:	       118 ns/op
BenchmarkAll/3._Big_cache_10’000_events/3.1._Sequenced/Hashicorp-Seq-Put-10000-4      	     398	   3187714 ns/op	 1615289 B/op	   10288 allocs/op
BenchmarkAll/3._Big_cache_10’000_events/3.1._Sequenced/Hashicorp-Seq-Get-10000-4      	    1258	    897341 ns/op	       0 B/op	       0 allocs/op
	— Hashicorp:	 (Sequenced)	 Put:	       318 ns/op; Get:	        89 ns/op
BenchmarkAll/3._Big_cache_10’000_events/3.1._Sequenced/Theine-Seq-Put-10000-4         	     208	   6013545 ns/op	 2241917 B/op	   10722 allocs/op
BenchmarkAll/3._Big_cache_10’000_events/3.1._Sequenced/Theine-Seq-Get-10000-4         	     496	   2614798 ns/op	      18 B/op	       0 allocs/op
	— Theine:	 (Sequenced)	 Put:	       601 ns/op; Get:	       261 ns/op
BenchmarkAll/3._Big_cache_10’000_events/3.2._Parallel/Hashicorp-Par-Put-500-20-4      	     258	   5069605 ns/op	 1617304 B/op	   10331 allocs/op
BenchmarkAll/3._Big_cache_10’000_events/3.2._Parallel/Hashicorp-Par-Get-500-20-4      	     422	   2568238 ns/op	    1777 B/op	      41 allocs/op
	— Hashicorp:	 (Parallel)	 Put:	       506 ns/op; Get:	       256 ns/op
BenchmarkAll/3._Big_cache_10’000_events/3.2._Parallel/Theine-Par-Put-500-20-4         	     307	   4701337 ns/op	 2244061 B/op	   10764 allocs/op
BenchmarkAll/3._Big_cache_10’000_events/3.2._Parallel/Theine-Par-Get-500-20-4         	    1557	   1030962 ns/op	    2563 B/op	      45 allocs/op
	— Theine:	 (Parallel)	 Put:	       470 ns/op; Get:	       103 ns/op
BenchmarkAll/4._Large_cache_100’000_events/3.1._Sequenced/Hashicorp-Seq-Put-100000-4           	      37	  65677443 ns/op	14490362 B/op	  104015 allocs/op
BenchmarkAll/4._Large_cache_100’000_events/3.1._Sequenced/Hashicorp-Seq-Get-100000-4           	      68	  18812243 ns/op	       0 B/op	       0 allocs/op
	— Hashicorp:	 (Sequenced)	 Put:	       656 ns/op; Get:	       188 ns/op
BenchmarkAll/4._Large_cache_100’000_events/3.1._Sequenced/Theine-Seq-Put-100000-4              	      16	  64079738 ns/op	19865739 B/op	  104134 allocs/op
BenchmarkAll/4._Large_cache_100’000_events/3.1._Sequenced/Theine-Seq-Get-100000-4              	      34	  34799138 ns/op	     265 B/op	       1 allocs/op
	— Theine:	 (Sequenced)	 Put:	       640 ns/op; Get:	       347 ns/op
BenchmarkAll/4._Large_cache_100’000_events/3.2._Parallel/Hashicorp-Par-Put-1000-100-4          	      25	  68356940 ns/op	14501165 B/op	  104226 allocs/op
BenchmarkAll/4._Large_cache_100’000_events/3.2._Parallel/Hashicorp-Par-Get-1000-100-4          	      28	  37938100 ns/op	    9090 B/op	     203 allocs/op
	— Hashicorp:	 (Parallel)	 Put:	       683 ns/op; Get:	       379 ns/op
BenchmarkAll/4._Large_cache_100’000_events/3.2._Parallel/Theine-Par-Put-1000-100-4             	      33	 120494927 ns/op	19889985 B/op	  104366 allocs/op
BenchmarkAll/4._Large_cache_100’000_events/3.2._Parallel/Theine-Par-Get-1000-100-4             	     100	  10171516 ns/op	   27345 B/op	     313 allocs/op
	— Theine:	 (Parallel)	 Put:	      1204 ns/op; Get:	       101 ns/op
PASS
ok  	github.com/voedger/voedger/pkg/objcache	110.138s
```

```text
Running tool: D:\Go\bin\go.exe test -benchmem -run=^$ -bench ^BenchmarkCacheParallelism$ github.com/voedger/voedger/pkg/objcache

goos: windows
goarch: amd64
pkg: github.com/voedger/voedger/pkg/objcache
cpu: Intel(R) Core(TM) i5-3570 CPU @ 3.40GHz
BenchmarkCacheParallelism/Hashicorp-Par-Put-1000-1-4         	    3870	    310085 ns/op	  187192 B/op	    1051 allocs/op
BenchmarkCacheParallelism/Hashicorp-Par-Get-1000-1-4         	   13912	     86115 ns/op	     104 B/op	       3 allocs/op
	— Hashicorp:	 (Parallel)	 Put:	       310 ns/op; Get:	        86 ns/op
BenchmarkCacheParallelism/Theine-Par-Put-1000-1-4            	    1058	   1085792 ns/op	  264784 B/op	    1481 allocs/op
BenchmarkCacheParallelism/Theine-Par-Get-1000-1-4            	    4999	    248679 ns/op	     105 B/op	       3 allocs/op
	— Theine:	 (Parallel)	 Put:	      1085 ns/op; Get:	       248 ns/op
BenchmarkCacheParallelism/Hashicorp-Par-Put-1000-11-4        	     211	   4816381 ns/op	 1688306 B/op	   11350 allocs/op
BenchmarkCacheParallelism/Hashicorp-Par-Get-1000-11-4        	     398	   2918865 ns/op	    1013 B/op	      23 allocs/op
	— Hashicorp:	 (Parallel)	 Put:	       437 ns/op; Get:	       265 ns/op
BenchmarkCacheParallelism/Theine-Par-Put-1000-11-4           	     253	   5226777 ns/op	 2348518 B/op	   11791 allocs/op
BenchmarkCacheParallelism/Theine-Par-Get-1000-11-4           	    1062	   1065135 ns/op	    1913 B/op	      25 allocs/op
	— Theine:	 (Parallel)	 Put:	       475 ns/op; Get:	        96 ns/op
BenchmarkCacheParallelism/Hashicorp-Par-Put-1000-21-4        	     100	  10905279 ns/op	 3295243 B/op	   21644 allocs/op
BenchmarkCacheParallelism/Hashicorp-Par-Get-1000-21-4        	     176	   6937866 ns/op	    1894 B/op	      43 allocs/op
	— Hashicorp:	 (Parallel)	 Put:	       519 ns/op; Get:	       330 ns/op
BenchmarkCacheParallelism/Theine-Par-Put-1000-21-4           	     147	   8232852 ns/op	 4545900 B/op	   22087 allocs/op
BenchmarkCacheParallelism/Theine-Par-Get-1000-21-4           	     651	   1962346 ns/op	    4488 B/op	      60 allocs/op
	— Theine:	 (Parallel)	 Put:	       392 ns/op; Get:	        93 ns/op
BenchmarkCacheParallelism/Hashicorp-Par-Put-1000-31-4        	      82	  18428568 ns/op	 5859436 B/op	   32209 allocs/op
BenchmarkCacheParallelism/Hashicorp-Par-Get-1000-31-4        	     100	  11332895 ns/op	    2785 B/op	      63 allocs/op
	— Hashicorp:	 (Parallel)	 Put:	       594 ns/op; Get:	       365 ns/op
BenchmarkCacheParallelism/Theine-Par-Put-1000-31-4           	     100	  12441000 ns/op	 7455483 B/op	   32654 allocs/op
BenchmarkCacheParallelism/Theine-Par-Get-1000-31-4           	     424	   2792709 ns/op	    7285 B/op	      84 allocs/op
	— Theine:	 (Parallel)	 Put:	       401 ns/op; Get:	        90 ns/op
BenchmarkCacheParallelism/Hashicorp-Par-Put-1000-41-4        	      62	  21278156 ns/op	 6509532 B/op	   42272 allocs/op
BenchmarkCacheParallelism/Hashicorp-Par-Get-1000-41-4        	      79	  15145216 ns/op	    3624 B/op	      83 allocs/op
	— Hashicorp:	 (Parallel)	 Put:	       518 ns/op; Get:	       369 ns/op
BenchmarkCacheParallelism/Theine-Par-Put-1000-41-4           	      69	  15266887 ns/op	 8946547 B/op	   42693 allocs/op
BenchmarkCacheParallelism/Theine-Par-Get-1000-41-4           	     337	   3550960 ns/op	    9306 B/op	      92 allocs/op
	— Theine:	 (Parallel)	 Put:	       372 ns/op; Get:	        86 ns/op
BenchmarkCacheParallelism/Hashicorp-Par-Put-1000-51-4        	      60	  27181523 ns/op	 7335544 B/op	   53183 allocs/op
BenchmarkCacheParallelism/Hashicorp-Par-Get-1000-51-4        	      60	  18628932 ns/op	    4612 B/op	     104 allocs/op
	— Hashicorp:	 (Parallel)	 Put:	       532 ns/op; Get:	       365 ns/op
BenchmarkCacheParallelism/Theine-Par-Put-1000-51-4           	      74	  26010997 ns/op	10308183 B/op	   53529 allocs/op
BenchmarkCacheParallelism/Theine-Par-Get-1000-51-4           	     258	   4505719 ns/op	   13597 B/op	     143 allocs/op
	— Theine:	 (Parallel)	 Put:	       510 ns/op; Get:	        88 ns/op
BenchmarkCacheParallelism/Hashicorp-Par-Put-1000-61-4        	      38	  73477937 ns/op	11654616 B/op	   63478 allocs/op
BenchmarkCacheParallelism/Hashicorp-Par-Get-1000-61-4        	      51	  23387878 ns/op	    5464 B/op	     123 allocs/op
	— Hashicorp:	 (Parallel)	 Put:	      1204 ns/op; Get:	       383 ns/op
BenchmarkCacheParallelism/Theine-Par-Put-1000-61-4           	      49	  42310961 ns/op	14775931 B/op	   63839 allocs/op
BenchmarkCacheParallelism/Theine-Par-Get-1000-61-4           	     186	   6166116 ns/op	   13603 B/op	     142 allocs/op
	— Theine:	 (Parallel)	 Put:	       693 ns/op; Get:	       101 ns/op
BenchmarkCacheParallelism/Hashicorp-Par-Put-1000-71-4        	      28	 139990068 ns/op	12293104 B/op	   73487 allocs/op
BenchmarkCacheParallelism/Hashicorp-Par-Get-1000-71-4        	      39	  27671395 ns/op	    6468 B/op	     145 allocs/op
	— Hashicorp:	 (Parallel)	 Put:	      1971 ns/op; Get:	       389 ns/op
BenchmarkCacheParallelism/Theine-Par-Put-1000-71-4           	      49	  54240927 ns/op	16785685 B/op	   73850 allocs/op
BenchmarkCacheParallelism/Theine-Par-Get-1000-71-4           	     156	   7090469 ns/op	   13839 B/op	     143 allocs/op
	— Theine:	 (Parallel)	 Put:	       763 ns/op; Get:	        99 ns/op
BenchmarkCacheParallelism/Hashicorp-Par-Put-1000-81-4        	      26	  71008046 ns/op	12938704 B/op	   83531 allocs/op
BenchmarkCacheParallelism/Hashicorp-Par-Get-1000-81-4        	      38	  32361082 ns/op	    7425 B/op	     165 allocs/op
	— Hashicorp:	 (Parallel)	 Put:	       876 ns/op; Get:	       399 ns/op
BenchmarkCacheParallelism/Theine-Par-Put-1000-81-4           	      22	  60169568 ns/op	17749574 B/op	   83877 allocs/op
BenchmarkCacheParallelism/Theine-Par-Get-1000-81-4           	     124	   8354519 ns/op	   16687 B/op	     163 allocs/op
	— Theine:	 (Parallel)	 Put:	       742 ns/op; Get:	       103 ns/op
BenchmarkCacheParallelism/Hashicorp-Par-Put-1000-91-4        	      25	  85583956 ns/op	13740915 B/op	   94328 allocs/op
BenchmarkCacheParallelism/Hashicorp-Par-Get-1000-91-4        	      36	  36099803 ns/op	    8128 B/op	     184 allocs/op
	— Hashicorp:	 (Parallel)	 Put:	       940 ns/op; Get:	       396 ns/op
BenchmarkCacheParallelism/Theine-Par-Put-1000-91-4           	      37	 104631341 ns/op	18830781 B/op	   94456 allocs/op
BenchmarkCacheParallelism/Theine-Par-Get-1000-91-4           	     114	  11423564 ns/op	   16090 B/op	     183 allocs/op
	— Theine:	 (Parallel)	 Put:	      1149 ns/op; Get:	       125 ns/op
BenchmarkCacheParallelism/Hashicorp-Par-Put-1000-101-4       	      21	  81814210 ns/op	14583623 B/op	  105320 allocs/op
BenchmarkCacheParallelism/Hashicorp-Par-Get-1000-101-4       	      28	  41022696 ns/op	    9042 B/op	     204 allocs/op
	— Hashicorp:	 (Parallel)	 Put:	       810 ns/op; Get:	       406 ns/op
BenchmarkCacheParallelism/Theine-Par-Put-1000-101-4          	      28	 100510575 ns/op	20001274 B/op	  105483 allocs/op
BenchmarkCacheParallelism/Theine-Par-Get-1000-101-4          	     100	  10702592 ns/op	   30540 B/op	     339 allocs/op
	— Theine:	 (Parallel)	 Put:	       995 ns/op; Get:	       105 ns/op
PASS
ok  	github.com/voedger/voedger/pkg/objcache	276.279s
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