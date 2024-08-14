/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package journal

import (
	"time"

	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys"
)

var wLogDatesProjector = func(event istructs.IPLogEvent, s istructs.IState, intents istructs.IIntents) (err error) {
	timestamp := time.UnixMilli(int64(event.RegisteredAt())).UTC()
	kb, err := s.KeyBuilder(sys.Storage_View, QNameViewWLogDates)
	if err != nil {
		return
	}
	kb.PutInt32(field_Year, int32(timestamp.Year()))
	kb.PutInt32(field_DayOfYear, int32(timestamp.YearDay()))

	sv, ok, err := s.CanExist(kb)
	if err != nil {
		return
	}

	lo := int64(event.WLogOffset())
	fo := lo
	if ok {
		if sv.AsInt64(field_LastOffset) >= lo {
			// skip for idempotency
			return nil
		}
		fo = sv.AsInt64(field_FirstOffset)
	}

	vb, err := intents.NewValue(kb)
	if err != nil {
		return
	}
	vb.PutInt64(field_FirstOffset, fo)
	vb.PutInt64(field_LastOffset, lo)
	return
}
