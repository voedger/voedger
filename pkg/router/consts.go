/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package router

import (
	"net/http"
	"time"

	coreutils "github.com/voedger/voedger/pkg/utils"
)

const (
	HTTPSPort                       = 443
	DefaultACMEServerReadTimeout    = 5 * time.Second
	DefaultACMEServerWriteTimeout   = 5 * time.Second
	subscriptionsCloseCheckInterval = 100 * time.Millisecond
	localhost                       = "127.0.0.1"
	parseInt64Base                  = 10
	parseInt64Bits                  = 64
	wsid                            = "wsid"
	appOwner                        = "appOwner"
	appName                         = "appName"
	blobID                          = "blobID"
	resourceName                    = "resourceName"
	//Timeouts should be greater than NATS timeouts to proper use in browser(multiply responses)
	DefaultReadTimeout      = 15
	DefaultWriteTimeout     = 15
	DefaultConnectionsLimit = 10000
)

var (
	bearerPrefixLen = len(coreutils.BearerPrefix)
	// airsBPPartitionsAmount int                         = 100 // changes in tests
	onRequestCtxClosed  func()                      = nil // used in tests
	onAfterSectionWrite func(w http.ResponseWriter) = nil // used in tests
)
