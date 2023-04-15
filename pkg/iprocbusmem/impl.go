/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package iprocbusmem

import (
	"github.com/untillpro/voedger/pkg/iprocbus"
)

type implIProcBus struct {
	// protected by airs-bp3 lifecycle: write only on wiring stage, read only after start
	chans [][]iprocbus.ServiceChannel
}

func (b *implIProcBus) ServiceChannel(groupIdx int, channelIdx int) (res iprocbus.ServiceChannel) {
	return b.chans[groupIdx][channelIdx]
}

func (b *implIProcBus) Submit(groupIdx int, channelIdx int, msg interface{}) (ok bool) {
	select {
	case b.chans[groupIdx][channelIdx] <- msg:
		return true
	default:
		return false
	}
}
