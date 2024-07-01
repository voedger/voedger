/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package router

import (
	"time"

	coreutils "github.com/voedger/voedger/pkg/utils"
)

const (
	HTTPSPort                       = 443
	DefaultACMEServerReadTimeout    = 5 * time.Second
	DefaultACMEServerWriteTimeout   = 5 * time.Second
	subscriptionsCloseCheckInterval = 100 * time.Millisecond
	DefaultPort                     = 8822
	DefaultConnectionsLimit         = 10000
	DefaultRouterReadTimeout        = 15
	DefaultRouterWriteTimeout       = 15
	localhost                       = "127.0.0.1"
	parseInt64Base                  = 10
	parseInt64Bits                  = 64
	WSID                            = "wsid"
	AppOwner                        = "appOwner"
	AppName                         = "appName"
	blobID                          = "blobID"
	ResourceName                    = "resourceName"
	hours24                         = 24 * time.Hour
)

var (
	bearerPrefixLen           = len(coreutils.BearerPrefix)
	onRequestCtxClosed func() = nil // used in tests
	elem1                     = map[string]interface{}{"fld1": "fld1Val"}
	adminEndpoint             = "127.0.0.1:55555"
)
