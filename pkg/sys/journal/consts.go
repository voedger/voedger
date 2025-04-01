/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package journal

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/extensionpoints"
)

const (
	field_Year               = "Year"
	field_DayOfYear          = "DayOfYear"
	field_FirstOffset        = "FirstOffset"
	field_LastOffset         = "LastOffset"
	Field_From               = "From"
	Field_Till               = "Till"
	Field_EventTypes         = "EventTypes"
	field_IndexForTimestamps = "IndexForTimestamps"
	field_RangeUnit          = "RangeUnit"
	Field_Offset             = "Offset"
	Field_EventTime          = "EventTime"
	Field_Event              = "Event"
)

const (
	rangeUnit_UnixTimestamp                       = "UnixTimestamp"
	rangeUnit_Offset                              = "Offset"
	EPJournalIndices        extensionpoints.EPKey = "JournalIndices"
	EPJournalPredicates     extensionpoints.EPKey = "JournalPredicates"
)

var (
	QNameViewWLogDates      = appdef.NewQName(appdef.SysPackage, "WLogDates")
	QNameProjectorWLogDates = appdef.NewQName(appdef.SysPackage, "ProjectorWLogDates")
)
