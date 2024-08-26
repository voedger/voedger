/*
  - Copyright (c) 2024-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/
package storages

import (
	"time"

	"github.com/voedger/voedger/pkg/appdef"
)

const (
	defaultHTTPClientTimeout              = 20_000 * time.Millisecond
	httpStorageKeyBuilderStringerSliceCap = 3
	field_WSKind                          = "WSKind"
	wsidTypeValidatorCacheSize            = 100
)

var (
	qNameCDocWorkspaceDescriptor = appdef.NewQName(appdef.SysPackage, "WorkspaceDescriptor")
)
