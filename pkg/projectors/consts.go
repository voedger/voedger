/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package projectors

import (
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

var (
	qnameProjectionOffsets       = appdef.NewQName(appdef.SysPackage, "projectionOffsets")
	cudTypeKindToGlobalDocQNames = map[appdef.TypeKind][]appdef.QName{
		appdef.TypeKind_CDoc:    {istructs.QNameCDoc, istructs.QNameCRecord},
		appdef.TypeKind_WDoc:    {istructs.QNameWDoc, istructs.QNameWRecord},
		appdef.TypeKind_ODoc:    {istructs.QNameODoc, istructs.QNameORecord},
		appdef.TypeKind_CRecord: {istructs.QNameCRecord},
		appdef.TypeKind_WRecord: {istructs.QNameWRecord},
		appdef.TypeKind_ORecord: {istructs.QNameORecord},
	}
)

const (
	partitionFld     = "partition"
	projectorNameFld = "projector"
	offsetFld        = "offset"
)

const (
	defaultIntentsLimit          = 100
	defaultBundlesLimit          = 100
	defaultFlushInterval         = time.Millisecond * 100
	defaultFlushPositionInterval = time.Minute
	actualizerErrorDelay         = time.Second * 30
	n10nChannelDuration          = 100 * 365 * 24 * time.Hour
)

var PLogUpdatesQName = appdef.NewQName(appdef.SysPackage, "PLogUpdates")

// size of a log piece (number of events) read by the actualizer at one time
const plogReadBatchSize = 50
