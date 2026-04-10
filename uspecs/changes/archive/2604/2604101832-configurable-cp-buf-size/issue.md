# AIR-3544: Make command processor channel buffer size configurable (default 0)

- **Type:** Sub-task
- **Status:** In Progress
- **Assignee:** d.gribanov@dev.untill.com

## Problem

The command processor channel buffer size is hardcoded to `10` (`DefaultNumCommandProcessors`) in `pkg/vvm/provide.go`:

```go
iprocbusmem.ChannelGroup{
    NumChannels:       uint(vvmCfg.NumCommandProcessors),
    ChannelBufferSize: uint(DefaultNumCommandProcessors), // hardcoded to 10
}
```

`procbus.Submit` is completely non-blocking:

```go
func (b *implIProcBus) Submit(groupIdx uint, channelIdx uint, msg interface{}) (ok bool) {
    select {
    case b.chans[groupIdx][channelIdx] <- msg:
        return true
    default:
        return false
    }
}
```

With `ChannelBufferSize: 10`, up to 10 commands can silently queue inside the channel buffer before `Submit` starts returning `false`. Each queued command holds a goroutine (the HTTP handler goroutine calling `bus.SendRequest` is blocked waiting for the response). During the AIR-3536 outage this caused commands to wait up to an hour in the buffer — far past the point when the originating HTTP request was cancelled and the client had given up.

All other processor channels already use `ChannelBufferSize: 0`:

```go
// query v1
iprocbusmem.ChannelGroup{NumChannels: 1, ChannelBufferSize: 0}

// query v2
iprocbusmem.ChannelGroup{NumChannels: 1, ChannelBufferSize: 0}

// BLOB
iprocbusmem.ChannelGroup{NumChannels: 1, ChannelBufferSize: 0}
```

With `buffer=0`, `Submit` returns `false` the instant the processor is busy. The VVM request handler immediately sends `503 Service Unavailable` back to the caller. This is the correct fail-fast behaviour.

## Solution

- Add `CommandProcessorChannelBufferSize` field to `VVMConfig` (type `uint`)
- Use it instead of the hardcoded value when provisioning the command processor channel group in `provide.go`
- Default value: `0` (consistent with all other processor channels, fail-fast behaviour)
- Scope: voedger repository, `pkg/vvm` (`VVMConfig`, `provide.go`, `consts.go`)
