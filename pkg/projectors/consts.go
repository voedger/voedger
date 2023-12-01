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
	qnameProjectionOffsets   = appdef.NewQName(appdef.SysPackage, "projectionOffsets")
	typeKindToGlobalDocQName = map[appdef.TypeKind]appdef.QName{
		appdef.TypeKind_CDoc:    istructs.QNameCDoc,
		appdef.TypeKind_WDoc:    istructs.QNameWDoc,
		appdef.TypeKind_ODoc:    istructs.QNameODoc,
		appdef.TypeKind_CRecord: istructs.QNameCRecord,
		appdef.TypeKind_WRecord: istructs.QNameWRecord,
		appdef.TypeKind_ORecord: istructs.QNameORecord,
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
