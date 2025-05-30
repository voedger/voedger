/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package router

import (
	"time"
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
	URLPlaceholder_wsid             = "wsid"
	URLPlaceholder_appOwner         = "appOwner"
	URLPlaceholder_appName          = "appName"
	URLPlaceholder_blobIDOrSUUID    = "blobIDOrSUUID"
	URLPlaceholder_resourceName     = "resourceName"
	URLPlaceholder_pkg              = "pkg"
	URLPlaceholder_table            = "table"
	URLPlaceholder_id               = "id"
	URLPlaceholder_command          = "command"
	URLPlaceholder_query            = "query"
	URLPlaceholder_view             = "view"
	URLPlaceholder_workspace        = "workspace"
	URLPlaceholder_rolePkg          = "rolePkg"
	URLPlaceholder_role             = "role"
	hours24                         = 24 * time.Hour
	DefaultRetryAfterSecondsOn503   = 1
)

var (
	onRequestCtxClosed func() = nil // used in tests
	adminEndpoint             = "127.0.0.1:55555"
)

const (
	fieldLogin       = "login"
	fieldPassword    = "password"
	fieldDisplayName = "displayName"
)
