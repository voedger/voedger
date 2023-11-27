/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package ihttpimpl

import "time"

const (
	defaultReadHeaderTimeout = time.Second
	staticPath               = "/static/"
)

type contextKey int

const (
	varsKey contextKey = iota
	routeKey
)
