[![codecov](https://codecov.io/gh/heeus/core-logger/branch/main/graph/badge.svg?token=R8903H0E1V)](https://codecov.io/gh/heeus/core-logger)
# logger

Simple go logger with logging level. Default output will be like this:

```
09/29 13:29:04.355: *****: [core-logger.Test_BasicUsage:22]: Hello world arg1 arg2
09/29 13:29:04.373: !!!: [core-logger.Test_BasicUsage:23]: My warning
09/29 13:29:04.374: ===: [core-logger.Test_BasicUsage:24]: My info
09/29 13:29:04.374: ---: [core-logger.Test_BasicUsage:35]: Now you should see my Trace
09/29 13:29:04.374: !!!: [core-logger.Test_BasicUsage:41]: You should see my warning
09/29 13:29:04.374: !!!: [core-logger.Test_BasicUsage:42]: You should see my info
09/29 13:29:04.374: *****: [core-logger.(*mystruct).iWantToLog:55]: OOPS
```

See [impl_test.Test_BasicUsage](impl_test.go#L24) for examples

# Links

- [Why does the TRACE level exist, and when should I use it rather than DEBUG?](https://softwareengineering.stackexchange.com/questions/279690/why-does-the-trace-level-exist-and-when-should-i-use-it-rather-than-debug)
  - [Good answer](https://softwareengineering.stackexchange.com/questions/279690/why-does-the-trace-level-exist-and-when-should-i-use-it-rather-than-debug/360810#360810)
