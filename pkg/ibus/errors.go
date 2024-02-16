/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package ibus

import (
	"errors"
	"net/http"
)

var ErrReadTimeoutExpired = errors.New("ibus.ErrReadTimeoutExpired")
var ErrSlowClient = errors.New("ibus.ErrSlowClient")
var ErrClientClosedRequest = errors.New("ibus.ClientClosedRequest")
var ErrReceiverNotFound = errors.New("ibus.ErrReceiverNotFound")
var ErrServiceUnavailable = errors.New("ibus.ErrServiceUnavailable")
var ErrBusUnavailable = errors.New("ibus.ErrBusUnavailable")

var ErrStatuses = map[error]int{

	ErrClientClosedRequest: StatusClientClosedRequest,
	ErrReadTimeoutExpired:  http.StatusGatewayTimeout,
	ErrReceiverNotFound:    http.StatusBadRequest,

	// Better choice would be StatusResponseTimeout but it does not exist
	ErrSlowClient: http.StatusGatewayTimeout, // TODO better choice???

	ErrServiceUnavailable: http.StatusServiceUnavailable,
	ErrBusUnavailable:     http.StatusServiceUnavailable,
}
