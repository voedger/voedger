/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package projectors

import (
	"time"

	"github.com/voedger/voedger/pkg/istructs"
)

var (
	qnameProjectionOffsets               = istructs.NewQName(istructs.SysPackage, "projectionOffsets")
	qnameProjectionOffsetsPartitionKey   = istructs.NewQName(istructs.SysPackage, "projectionOffsetsKey")
	qnameProjectionOffsetsClusteringCols = istructs.NewQName(istructs.SysPackage, "projectionOffsetsSort")
	qnameProjectionOffsetsValue          = istructs.NewQName(istructs.SysPackage, "projectionOffsetsValue")
)

const (
	partitionFld     = "partition"
	projectorNameFld = "projector"
	offsetFld        = "offset"
)

const (
	defaultIntentsLimit  = 100
	defaultBundlesLimit  = 100
	defaultFlushInterval = time.Millisecond * 100
	actualizerErrorDelay = time.Second * 30
	n10nChannelDuration  = 100 * 365 * 24 * time.Hour
)

var PlogQName = istructs.NewQName(istructs.SysPackage, "PLog")
