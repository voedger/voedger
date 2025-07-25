/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package actualizers

import (
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/sys"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/sys/builtin"
)

var (
	qnameProjectionOffsets = sys.ProjectionOffsetsView.Name
	partitionFld           = sys.ProjectionOffsetsView.Fields.Partition
	projectorNameFld       = sys.ProjectionOffsetsView.Fields.Projector
	offsetFld              = sys.ProjectionOffsetsView.Fields.Offset
	actualizerErrorDelay   = defaultActualizerErrorDelay // 30 seconds in production, 100ms in tests
)

const (
	defaultIntentsLimit          = 100
	defaultBundlesLimit          = 100
	defaultFlushInterval         = time.Millisecond * 100
	defaultFlushPositionInterval = time.Minute
	defaultActualizerErrorDelay  = time.Second * 30
	testActualizerErrorDelay     = 100 * time.Millisecond
	n10nChannelDuration          = 100 * 365 * 24 * time.Hour
	borrowRetryDelay             = 50 * time.Millisecond
	initFailureErrorLogInterval  = 30 * time.Second
	DefaultIntentsLimit          = builtin.MaxCUDs * 10
)

func init() {
	if coreutils.IsTest() {
		actualizerErrorDelay = testActualizerErrorDelay
	}
}

var PLogUpdatesQName = appdef.NewQName(appdef.SysPackage, "PLogUpdates")

// size of a batch (maximum number of events) to read by the actualizer from PLog at one time
const plogReadBatchSize = 50
