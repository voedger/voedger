/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package journal

import (
	"github.com/voedger/voedger/pkg/appdef"
)

const (
	field_Year               = "Year"
	field_DayOfYear          = "DayOfYear"
	field_FirstOffset        = "FirstOffset"
	field_LastOffset         = "LastOffset"
	field_From               = "From"
	field_Till               = "Till"
	Field_EventTypes         = "EventTypes"
	field_IndexForTimestamps = "IndexForTimestamps"
	field_RangeUnit          = "RangeUnit"
	Field_Offset             = "Offset"
	Field_EventTime          = "EventTime"
	Field_Event              = "Event"
)

const (
	rangeUnit_UnixTimestamp = "UnixTimestamp"
	rangeUnit_Offset        = "Offset"
)

var QNameViewWLogDates = appdef.NewQName(appdef.SysPackage, "WLogDates")
