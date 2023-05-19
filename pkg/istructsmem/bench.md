
# 14.11.2021 Results

https://github.com/untillpro/voedger/pkg/istructsmem/tree/36162aff8055b14481b5daedcb029d6e7b829c87

```go
cpu: Intel(R) Core(TM) i5-3570 CPU @ 3.40GHz
Benchmark_BuildRawEvent/numOfIntFields=2
  444417	      2579 ns/op	    1320 B/op	      27 allocs/op
Benchmark_BuildRawEvent/numOfIntFields=4
  285697	      4221 ns/op	    1861 B/op	      36 allocs/op
Benchmark_BuildRawEvent/numOfIntFields=8
  169003	      7130 ns/op	    3047 B/op	      53 allocs/op
Benchmark_BuildRawEvent/numOfIntFields=16
   96768	     12660 ns/op	    5455 B/op	      87 allocs/op
Benchmark_BuildRawEvent/numOfIntFields=32
   50844	     23819 ns/op	   10756 B/op	     154 allocs/op
```

```go
Benchmark_UnmarshallJSONForBuildRawEvent/numOfIntFields=2
  387073	      2912 ns/op	     736 B/op	      23 allocs/op
Benchmark_UnmarshallJSONForBuildRawEvent/numOfIntFields=4
  235280	      5177 ns/op	     968 B/op	      41 allocs/op
Benchmark_UnmarshallJSONForBuildRawEvent/numOfIntFields=8
  106188	     11433 ns/op	    3244 B/op	      79 allocs/op
Benchmark_UnmarshallJSONForBuildRawEvent/numOfIntFields=16
   52860	     22665 ns/op	    6727 B/op	     159 allocs/op
Benchmark_UnmarshallJSONForBuildRawEvent/numOfIntFields=32
   26256	     45059 ns/op	   13925 B/op	     322 allocs/op
```

# 15.12.2021 Results

https://github.com/untillpro/voedger/pkg/istructsmem/tree/0bfaaed1070f549dcdc1828076c6830311dcc572

```go
cpu: Intel(R) Core(TM) i7-3770 CPU @ 3.40GHz
Benchmark_BuildRawEvent/numOfFields=4-8
  196881	      6013 ns/op	    166303 op/s	    2433 B/op	      45 allocs/op
Benchmark_BuildRawEvent/numOfFields=8-8
  167079	      7109 ns/op	    140673 op/s	    3025 B/op	      55 allocs/op
Benchmark_BuildRawEvent/numOfFields=16-8
  113571	     10349 ns/op	     96626 op/s	    4466 B/op	      75 allocs/op
Benchmark_BuildRawEvent/numOfFields=32-8
   79992	     16462 ns/op	     60746 op/s	    5523 B/op	     107 allocs/op
Benchmark_BuildRawEvent/numOfFields=64-8
   43762	     26933 ns/op	     37129 op/s	    9207 B/op	     173 allocs/op
```

```go
Benchmark_UnmarshallJSONForBuildRawEvent/numOfFields=4-8
  440408	      2897 ns/op	    345192 op/s	     736 B/op	      23 allocs/op
Benchmark_UnmarshallJSONForBuildRawEvent/numOfFields=8-8
  250095	      4887 ns/op	    204638 op/s	     968 B/op	      41 allocs/op
Benchmark_UnmarshallJSONForBuildRawEvent/numOfFields=16-8
  111654	     10894 ns/op	     91794 op/s	    3246 B/op	      79 allocs/op
Benchmark_UnmarshallJSONForBuildRawEvent/numOfFields=32-8
   54631	     21926 ns/op	     45609 op/s	    6726 B/op	     159 allocs/op
Benchmark_UnmarshallJSONForBuildRawEvent/numOfFields=64-8
   27327	     43245 ns/op	     23124 op/s	   13925 B/op	     322 allocs/op
```