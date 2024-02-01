// Copyright (c) 2021-present Voedger Authors.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.
// @author Michael Saigachenko


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
