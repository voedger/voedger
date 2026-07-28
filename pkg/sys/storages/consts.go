/*
  - Copyright (c) 2024-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/
package storages

import (
	"time"
)

const (
	defaultHTTPClientTimeout   = 20_000 * time.Millisecond
	field_WSKind               = "WSKind"
	wsidTypeValidatorCacheSize = 100
)
