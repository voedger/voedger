/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package iprocbusmem

import (
	"github.com/untillpro/voedger/pkg/iprocbus"
)

type ChannelGroup struct {
	NumChannels       int
	ChannelBufferSize int
}

// Usage:
//   - Create IProcBus
//   - CommandProcessorsGroup: group0:{NumChannels:10, ChannelBufferSize: 10}
//   - One command processor - one channel
//   - QueryProcessorsGroup: group1:{NumChannels:1, ChannelBufferSize: 0}
//   - Wire IProcBus.ServiceChannel(...) to services
//   - Use Submit() to deliver messages to services
func Provide(groups []ChannelGroup) (bus iprocbus.IProcBus) {
	res := &implIProcBus{make([][]iprocbus.ServiceChannel, len(groups))}
	for i, group := range groups {
		res.chans[i] = make([]iprocbus.ServiceChannel, group.NumChannels)
		for j := 0; j < group.NumChannels; j++ {
			res.chans[i][j] = make(iprocbus.ServiceChannel, group.ChannelBufferSize)
		}
	}
	return res
}
