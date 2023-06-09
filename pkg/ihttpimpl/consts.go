/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package ihttpimpl

import "time"

const (
	NumOfAPIProcessors       = 1
	APIChannelBufferSize     = 10
	defaultReadHeaderTimeout = time.Second
	staticPath               = "/static/"
)
