/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 * @author Maxim Geraskin
 */

package ihttpimpl

import (
	"time"
)

const (
	defaultACMEServerReadTimeout  = 5 * time.Second
	defaultACMEServerWriteTimeout = 5 * time.Second
	defaultHTTPPort               = 80
	defaultReadHeaderTimeout      = time.Second
	staticPath                    = "/static/"
)
