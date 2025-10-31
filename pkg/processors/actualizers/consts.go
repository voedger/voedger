/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package actualizers

import (
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/sys"
	"github.com/voedger/voedger/pkg/sys/builtin"
)

var (
	qnameProjectionOffsets = sys.ProjectionOffsetsView.Name
	partitionFld           = sys.ProjectionOffsetsView.Fields.Partition
	projectorNameFld       = sys.ProjectionOffsetsView.Fields.Projector
	offsetFld              = sys.ProjectionOffsetsView.Fields.Offset
)

const (
	defaultIntentsLimit          = 100
	defaultBundlesLimit          = 100
	defaultFlushInterval         = time.Millisecond * 100
	defaultFlushPositionInterval = time.Minute
	defaultRetryInitialDelay     = 100 * time.Millisecond
	defaultRetryMaxDelay         = 3 * time.Minute
	n10nChannelDuration          = 100 * 365 * 24 * time.Hour
	DefaultIntentsLimit          = builtin.MaxCUDs * 10
)

var PLogUpdatesQName = appdef.NewQName(appdef.SysPackage, "PLogUpdates")

// size of a batch (maximum number of events) to read by the actualizer from PLog at one time
const plogReadBatchSize = 50
