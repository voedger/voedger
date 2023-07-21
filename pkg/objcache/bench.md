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
Running tool: D:\Go\bin\go.exe test -benchmem -run=^$ -bench ^BenchmarkParallelism$ github.com/voedger/voedger/pkg/objcache

goos: windows
goarch: amd64
pkg: github.com/voedger/voedger/pkg/objcache
cpu: Intel(R) Core(TM) i5-3570 CPU @ 3.40GHz
BenchmarkParallelism/Hashicorp-Par-Put-1000-1-4         	    3870	    304107 ns/op	  187223 B/op	    1051 allocs/op
BenchmarkParallelism/Hashicorp-Par-Get-1000-1-4         	   14096	     85275 ns/op	     104 B/op	       3 allocs/op
	— Hashicorp:	 (Parallel)	 Put:	       304 ns/op; Get:	        85 ns/op
BenchmarkParallelism/Theine-Par-Put-1000-1-4            	    1135	   1076775 ns/op	  264627 B/op	    1481 allocs/op
BenchmarkParallelism/Theine-Par-Get-1000-1-4            	    5216	    262736 ns/op	     105 B/op	       3 allocs/op
	— Theine:	 (Parallel)	 Put:	      1076 ns/op; Get:	       262 ns/op
BenchmarkParallelism/Hashicorp-Par-Put-1000-11-4        	     252	   4402625 ns/op	 1688130 B/op	   11349 allocs/op
BenchmarkParallelism/Hashicorp-Par-Get-1000-11-4        	     384	   3256798 ns/op	     999 B/op	      23 allocs/op
	— Hashicorp:	 (Parallel)	 Put:	       400 ns/op; Get:	       296 ns/op
BenchmarkParallelism/Theine-Par-Put-1000-11-4           	     313	   4926392 ns/op	 2348634 B/op	   11791 allocs/op
BenchmarkParallelism/Theine-Par-Get-1000-11-4           	    1335	    900098 ns/op	    1689 B/op	      23 allocs/op
	— Theine:	 (Parallel)	 Put:	       447 ns/op; Get:	        81 ns/op
BenchmarkParallelism/Hashicorp-Par-Put-1000-21-4        	     100	  10788186 ns/op	 3295206 B/op	   21645 allocs/op
BenchmarkParallelism/Hashicorp-Par-Get-1000-21-4        	     175	   6941478 ns/op	    1864 B/op	      43 allocs/op
	— Hashicorp:	 (Parallel)	 Put:	       513 ns/op; Get:	       330 ns/op
BenchmarkParallelism/Theine-Par-Put-1000-21-4           	     164	   7505013 ns/op	 4545641 B/op	   22087 allocs/op
BenchmarkParallelism/Theine-Par-Get-1000-21-4           	     603	   1744161 ns/op	    4335 B/op	      54 allocs/op
	— Theine:	 (Parallel)	 Put:	       357 ns/op; Get:	        83 ns/op
BenchmarkParallelism/Hashicorp-Par-Put-1000-31-4        	      74	  17409214 ns/op	 5858890 B/op	   32206 allocs/op
BenchmarkParallelism/Hashicorp-Par-Get-1000-31-4        	     100	  10739963 ns/op	    2800 B/op	      63 allocs/op
	— Hashicorp:	 (Parallel)	 Put:	       561 ns/op; Get:	       346 ns/op
BenchmarkParallelism/Theine-Par-Put-1000-31-4           	     100	  11400533 ns/op	 7456388 B/op	   32658 allocs/op
BenchmarkParallelism/Theine-Par-Get-1000-31-4           	     420	   2641144 ns/op	    5333 B/op	      66 allocs/op
	— Theine:	 (Parallel)	 Put:	       367 ns/op; Get:	        85 ns/op
BenchmarkParallelism/Hashicorp-Par-Put-1000-41-4        	      68	  24269440 ns/op	 6511267 B/op	   42281 allocs/op
BenchmarkParallelism/Hashicorp-Par-Get-1000-41-4        	      74	  16065284 ns/op	    3848 B/op	      83 allocs/op
	— Hashicorp:	 (Parallel)	 Put:	       591 ns/op; Get:	       391 ns/op
BenchmarkParallelism/Theine-Par-Put-1000-41-4           	      81	  22959815 ns/op	 8947063 B/op	   42695 allocs/op
BenchmarkParallelism/Theine-Par-Get-1000-41-4           	     345	   3589786 ns/op	    8468 B/op	      94 allocs/op
	— Theine:	 (Parallel)	 Put:	       559 ns/op; Get:	        87 ns/op
BenchmarkParallelism/Hashicorp-Par-Put-1000-51-4        	      56	  30972579 ns/op	 7335843 B/op	   53185 allocs/op
BenchmarkParallelism/Hashicorp-Par-Get-1000-51-4        	      60	  18945998 ns/op	    4616 B/op	     104 allocs/op
	— Hashicorp:	 (Parallel)	 Put:	       607 ns/op; Get:	       371 ns/op
BenchmarkParallelism/Theine-Par-Put-1000-51-4           	      68	  33997779 ns/op	10078009 B/op	   53523 allocs/op
BenchmarkParallelism/Theine-Par-Get-1000-51-4           	     266	   4769832 ns/op	   14820 B/op	     162 allocs/op
	— Theine:	 (Parallel)	 Put:	       666 ns/op; Get:	        93 ns/op
BenchmarkParallelism/Hashicorp-Par-Put-1000-61-4        	      36	 143335089 ns/op	11653362 B/op	   63475 allocs/op
BenchmarkParallelism/Hashicorp-Par-Get-1000-61-4        	      51	  23127312 ns/op	    5530 B/op	     124 allocs/op
	— Hashicorp:	 (Parallel)	 Put:	      2349 ns/op; Get:	       379 ns/op
BenchmarkParallelism/Theine-Par-Put-1000-61-4           	      56	  23616020 ns/op	14775651 B/op	   63838 allocs/op
BenchmarkParallelism/Theine-Par-Get-1000-61-4           	     180	   6490312 ns/op	   13746 B/op	     136 allocs/op
	— Theine:	 (Parallel)	 Put:	       387 ns/op; Get:	       106 ns/op
BenchmarkParallelism/Hashicorp-Par-Put-1000-71-4        	      31	  39662219 ns/op	12294222 B/op	   73494 allocs/op
BenchmarkParallelism/Hashicorp-Par-Get-1000-71-4        	      43	  27770860 ns/op	    6409 B/op	     144 allocs/op
	— Hashicorp:	 (Parallel)	 Put:	       558 ns/op; Get:	       391 ns/op
BenchmarkParallelism/Theine-Par-Put-1000-71-4           	      42	  43001879 ns/op	16785686 B/op	   73852 allocs/op
BenchmarkParallelism/Theine-Par-Get-1000-71-4           	     166	   7054639 ns/op	   18911 B/op	     143 allocs/op
	— Theine:	 (Parallel)	 Put:	       605 ns/op; Get:	        99 ns/op
BenchmarkParallelism/Hashicorp-Par-Put-1000-81-4        	      28	  56294214 ns/op	12937162 B/op	   83523 allocs/op
BenchmarkParallelism/Hashicorp-Par-Get-1000-81-4        	      38	  28528668 ns/op	    7514 B/op	     165 allocs/op
	— Hashicorp:	 (Parallel)	 Put:	       694 ns/op; Get:	       352 ns/op
BenchmarkParallelism/Theine-Par-Put-1000-81-4           	      31	  46955432 ns/op	17748844 B/op	   83873 allocs/op
BenchmarkParallelism/Theine-Par-Get-1000-81-4           	     135	   7533033 ns/op	   17840 B/op	     163 allocs/op
	— Theine:	 (Parallel)	 Put:	       579 ns/op; Get:	        93 ns/op
BenchmarkParallelism/Hashicorp-Par-Put-1000-91-4        	      27	  69859533 ns/op	13737047 B/op	   94310 allocs/op
BenchmarkParallelism/Hashicorp-Par-Get-1000-91-4        	      37	  34857916 ns/op	    8071 B/op	     183 allocs/op
	— Hashicorp:	 (Parallel)	 Put:	       767 ns/op; Get:	       383 ns/op
BenchmarkParallelism/Theine-Par-Put-1000-91-4           	      34	  61464003 ns/op	18831190 B/op	   94458 allocs/op
BenchmarkParallelism/Theine-Par-Get-1000-91-4           	     140	  10675201 ns/op	   19274 B/op	     183 allocs/op
	— Theine:	 (Parallel)	 Put:	       675 ns/op; Get:	       117 ns/op
PASS
ok  	github.com/voedger/voedger/pkg/objcache	216.587s
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