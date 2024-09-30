/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package journal

import (
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys"
)

func FindOffsetsByTimeRange(from, till time.Time, idx appdef.QName, s istructs.IState) (fo, lo int64, err error) {
	for y := from.Year(); y <= till.Year(); y++ {
		kb, err := s.KeyBuilder(sys.Storage_View, idx)
		if err != nil {
			return 0, 0, err
		}
		kb.PutInt32(field_Year, int32(y)) // nolint G115
		err = s.Read(kb, func(key istructs.IKey, value istructs.IStateValue) (err error) {
			yearDay := int(key.AsInt32(field_DayOfYear))
			if fo == int64(0) && (y > from.Year() || (y == from.Year() && yearDay >= from.YearDay())) {
				fo = value.AsInt64(field_FirstOffset)
			}
			if y < till.Year() || (y == till.Year() && yearDay <= till.YearDay()) {
				lo = value.AsInt64(field_LastOffset)
			}
			return
		})
		if err != nil {
			return 0, 0, err
		}
	}
	return
}
