/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package iprocbus

type ServiceChannel chan interface{}

// One per application
type IProcBus interface {
	// Used during wiring
	// This channel should be used by service to get its messages
	ServiceChannel(groupIdx int, channelIdx int) ServiceChannel

	// Message is submitted to the channel defined by groupIdx, channelIdx
	Submit(groupIdx int, channelIdx int, msg interface{}) (ok bool)
}
