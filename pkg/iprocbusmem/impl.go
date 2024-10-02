/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package iprocbusmem

import (
	"github.com/voedger/voedger/pkg/iprocbus"
)

type implIProcBus struct {
	// protected by VVM lifecycle: write only on wiring stage, read only after start
	chans [][]iprocbus.ServiceChannel
}

func (b *implIProcBus) ServiceChannel(groupIdx uint, channelIdx uint) (res iprocbus.ServiceChannel) {
	return b.chans[groupIdx][channelIdx]
}

func (b *implIProcBus) Submit(groupIdx uint, channelIdx uint, msg interface{}) (ok bool) {
	select {
	case b.chans[groupIdx][channelIdx] <- msg:
		return true
	default:
		return false
	}
}
