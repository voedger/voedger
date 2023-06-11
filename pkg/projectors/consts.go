/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package projectors

import (
	"time"

	"github.com/voedger/voedger/pkg/appdef"
)

var (
	qnameProjectionOffsets = appdef.NewQName(appdef.SysPackage, "projectionOffsets")
)

const (
	partitionFld     = "partition"
	projectorNameFld = "projector"
	offsetFld        = "offset"
)

const (
	defaultIntentsLimit           = 100
	defaultBundlesLimit           = 100
	defaultFlushInterval          = time.Millisecond * 100
	actualizerErrorDelay          = time.Second * 30
	n10nChannelDuration           = 100 * 365 * 24 * time.Hour
	autoSavePositionEveryNFlushes = 10
)

var PlogQName = appdef.NewQName(appdef.SysPackage, "PLog")
