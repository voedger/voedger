# fix TestRejectedCommandsDoNotRecoverOrWrite

- URL: https://untill.atlassian.net/browse/AIR-5301
- ID: AIR-5301
- State: In Progress⚒️
- Author: Denis Gribanov
- Labels: none
- Assignees: Denis Gribanov
- Linked issues: AIR-1640 — 🐞Internal technical problems

## Description

<details>

```text
 ==================
  WARNING: DATA RACE
  Write at 0x0000014b1440 by goroutine 1483:
    github.com/voedger/voedger/pkg/goutils/logger.SetCtxWriters()
        /home/runner/work/voedger/voedger/pkg/goutils/logger/loggerctx.go:18 +0x332
    github.com/voedger/voedger/pkg/goutils/logger.StartCapture()
        /home/runner/work/voedger/voedger/pkg/goutils/logger/logcapture.go:19 +0x153
    github.com/voedger/voedger/pkg/processors/command.TestRejectedCommandsDoNotRecoverOrWrite()
        /home/runner/work/voedger/voedger/pkg/processors/command/impl_test.go:2206 +0x499
    testing.tRunner()
        /opt/hostedtoolcache/go/1.27.4/x64/src/testing/testing.go:2195 +0x21c
    testing.(*T).Run.gowrap1()
        /opt/hostedtoolcache/go/1.27.4/x64/src/testing/testing.go:2260 +0x38

  Previous read at 0x0000014b1440 by goroutine 1485:
    github.com/voedger/voedger/pkg/goutils/logger.logCtx()
        /home/runner/work/voedger/voedger/pkg/goutils/logger/loggerctx.go:61 +0x97
    github.com/voedger/voedger/pkg/goutils/logger.InfoCtx()
        /home/runner/work/voedger/voedger/pkg/goutils/logger/loggerctx.go:40 +0x1e8
    github.com/voedger/voedger/pkg/in10nmem.(*n10nBroker).heartbeat30()
        /home/runner/work/voedger/voedger/pkg/in10nmem/impl.go:417 +0x112
    github.com/voedger/voedger/pkg/in10nmem.NewN10nBroker.gowrap2()
        /home/runner/work/voedger/voedger/pkg/in10nmem/provide.go:48 +0x54

  Goroutine 1483 (running) created at:
    testing.(*T).Run()
        /opt/hostedtoolcache/go/1.27.4/x64/src/testing/testing.go:2260 +0xb12
    testing.runTests.func1()
        /opt/hostedtoolcache/go/1.27.4/x64/src/testing/testing.go:2744 +0x84
    testing.tRunner()
        /opt/hostedtoolcache/go/1.27.4/x64/src/testing/testing.go:2195 +0x21c
    testing.runTests()
        /opt/hostedtoolcache/go/1.27.4/x64/src/testing/testing.go:2742 +0x9e9
    testing.(*M).Run()
        /opt/hostedtoolcache/go/1.27.4/x64/src/testing/testing.go:2602 +0xf44
    main.main()
        _testmain.go:96 +0x164

  Goroutine 1485 (running) created at:
    github.com/voedger/voedger/pkg/in10nmem.NewN10nBroker()
        /home/runner/work/voedger/voedger/pkg/in10nmem/provide.go:48 +0x6d2
    github.com/voedger/voedger/pkg/processors/command.setUp()
        /home/runner/work/voedger/voedger/pkg/processors/command/impl_test.go:2493 +0xe06
    github.com/voedger/voedger/pkg/processors/command.setUpRecoveryTestApp()
        /home/runner/work/voedger/voedger/pkg/processors/command/impl_test.go:699 +0xd0
    github.com/voedger/voedger/pkg/processors/command.TestRejectedCommandsDoNotRecoverOrWrite()
        /home/runner/work/voedger/voedger/pkg/processors/command/impl_test.go:2187 +0x190
    testing.tRunner()
        /opt/hostedtoolcache/go/1.27.4/x64/src/testing/testing.go:2195 +0x21c
    testing.(*T).Run.gowrap1()
        /opt/hostedtoolcache/go/1.27.4/x64/src/testing/testing.go:2260 +0x38
  ==================
  11/18 13:38:01.203: ===: [logger.SetLogLevelWithRestore.func1:39]: LogLevel restored to 3
  --- FAIL: TestRejectedCommandsDoNotRecoverOrWrite (0.01s)
  ```
  
  </details>
