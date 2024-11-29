/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package router

import (
	"time"

	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/iblobstorage"
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
	URLPlaceholder_WSID             = "wsid"
	URLPlaceholder_AppOwner         = "appOwner"
	URLPlaceholder_AppName          = "appName"
	URLPlaceholder_blobID           = "blobID"
	URLPlaceholder_ResourceName     = "resourceName"
	hours24                         = 24 * time.Hour
)

var (
	bearerPrefixLen                = len(coreutils.BearerPrefix)
	onRequestCtxClosed      func() = nil // used in tests
	elem1                          = map[string]interface{}{"fld1": "fld1Val"}
	adminEndpoint                  = "127.0.0.1:55555"
	durationToRegisterFuncs        = map[iblobstorage.DurationType]string{
		iblobstorage.DurationType_1Day: "c.sys.RegisterTempBLOB1d",
	}
)
