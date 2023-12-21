/*
*
* Copyright (c) 2021-present unTill Pro, Ltd.
*
* @author Michael Saigachenko
*
 */

package pipeline

// Error places
const (
	placeFlushDisassembling   = "flush-disassembling"
	placeFlushByTimer         = "flush-timer"
	placeCatchOnErr           = "catch-onErr"
	placeDoAsyncOutWorkIsNil  = "doAsync, outWork==nil"
	placeDoAsyncOutWorkNotNil = "doAsync, outWork!=nil"
	placeDoSync               = "doSync"
)
