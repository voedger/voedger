### Multiple selects takes time:

- Benchmark_QueryAndSend with reader: 1717 ns/op
- Benchmark_QueryAndSend with faster reader (fewer choices in select): 1468 ns/op

See also: [runtime: select on a shared channel is slow with many Ps](https://github.com/golang/go/issues/20351)
