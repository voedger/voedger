# voedger: fix SIGSEGV during tests

- URL: https://untill.atlassian.net/browse/AIR-4355
- ID: AIR-4355
- State: in-progress
- Author: Denis Gribanov
- Assignees: Denis Gribanov
- Labels: none

## Description

https://github.com/untillpro/airs-bp3/issues/2586

```text
unexpected fault address 0x7f7b39da2dcd
fatal error: fault
[signal SIGSEGV: segmentation violation code=0x1 addr=0x7f7b39da2dcd pc=0xcbdcc4]

goroutine 6359 gp=0xc0027ffc20 m=7 mp=0xc003efe008 [running]:
runtime.throw({0x1ef6ea0?, 0xc0036a7320?})
	runtime/panic.go:1229 +0x48
runtime.sigpanic()
	runtime/signal_unix.go:945 +0x285
github.com/google/flatbuffers/go.GetInt32(...)
	flatbuffers@v25.12.19+incompatible/go/encode.go:86
github.com/google/flatbuffers/go.(*Table).Offset(0xc002d01750, 0x4)
	flatbuffers@v25.12.19+incompatible/go/table.go:15 +0xc4
github.com/untillpro/dynobuffers.(*Buffer).getFieldUOffsetTByOrder(0xc002d01730, 0x0)
	dynobuffers@v0.0.0-20251212090544-93da105bf1da/dynobuffers.go:327 +0x58
github.com/untillpro/dynobuffers.(*Buffer).IterateFields(0xc002d01730, ...)
	dynobuffers@v0.0.0-20251212090544-93da105bf1da/dynobuffers.go:2138 +0x117
github.com/voedger/voedger/pkg/istructsmem.(*rowType).SpecifiedValues(0xc002b7f880, 0xc0036eed20)
	voedger/pkg/istructsmem/tables-types.go:149 +0x69c
github.com/voedger/voedger/pkg/coreutils.FieldsToMap(...)
	voedger/pkg/coreutils/objectreader.go:114 +0x491
github.com/voedger/voedger/pkg/processors.LogEventAndCUDs-range1(...)
	voedger/pkg/processors/utils.go:170 +0x350
github.com/voedger/voedger/pkg/istructsmem.(*cudType).enumRecs(...)
	voedger/pkg/istructsmem/event-types.go:454
github.com/voedger/voedger/pkg/istructsmem.(*eventType).CUDs(0xc002d1e288, 0xc00320fb00)
	voedger/pkg/istructsmem/event-types.go:293 +0xbf
github.com/voedger/voedger/pkg/processors.LogEventAndCUDs(...)
	voedger/pkg/processors/utils.go:159 +0xa71
github.com/voedger/voedger/pkg/processors/actualizers.logEventAndCUDs(...)
	voedger/pkg/processors/actualizers/async.go:476 +0x306
github.com/voedger/voedger/pkg/processors/actualizers.(*asyncProjector).DoAsync(...)
	voedger/pkg/processors/actualizers/async.go:439 +0x8c5
github.com/voedger/voedger/pkg/pipeline.(*WiredOperator).doAsync(...)
	voedger/pkg/pipeline/wired-operator.go:90 +0x103
github.com/voedger/voedger/pkg/pipeline.puller_async(0xc001d32930)
	voedger/pkg/pipeline/async.go:36 +0x46f
github.com/voedger/voedger/pkg/pipeline.NewAsyncPipeline.gowrap1()
	voedger/pkg/pipeline/async-pipeline-impl.go:54 +0x2f
runtime.goexit({})
	runtime/asm_amd64.s:1771 +0x1
created by github.com/voedger/voedger/pkg/pipeline.NewAsyncPipeline in goroutine 5582
```
