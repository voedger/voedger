/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package projectors

import (
	"time"

	"github.com/voedger/voedger/pkg/schemas"
)

var (
	qnameProjectionOffsets               = schemas.NewQName(schemas.SysPackage, "projectionOffsets")
	qnameProjectionOffsetsPartitionKey   = schemas.NewQName(schemas.SysPackage, "projectionOffsetsKey")
	qnameProjectionOffsetsClusteringCols = schemas.NewQName(schemas.SysPackage, "projectionOffsetsSort")
	qnameProjectionOffsetsValue          = schemas.NewQName(schemas.SysPackage, "projectionOffsetsValue")
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

var PlogQName = schemas.NewQName(schemas.SysPackage, "PLog")
