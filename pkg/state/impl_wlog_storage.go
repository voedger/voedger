/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"context"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

type wLogStorage struct {
	ctx        context.Context
	eventsFunc eventsFunc
	appDefFunc appDefFunc
	wsidFunc   WSIDFunc
}

func (s *wLogStorage) NewKeyBuilder(appdef.QName, istructs.IStateKeyBuilder) istructs.IStateKeyBuilder {
	return &wLogKeyBuilder{
		logKeyBuilder: logKeyBuilder{
			offset: istructs.FirstOffset,
			count:  1,
		},
		wsid: s.wsidFunc(),
	}
}
func (s *wLogStorage) Get(key istructs.IStateKeyBuilder) (value istructs.IStateValue, err error) {
	err = s.Read(key, func(_ istructs.IKey, v istructs.IStateValue) (err error) {
		value = v
		return nil
	})
	return value, err
}
func (s *wLogStorage) Read(kb istructs.IStateKeyBuilder, callback istructs.ValueCallback) (err error) {
	k := kb.(*wLogKeyBuilder)
	cb := func(wlogOffset istructs.Offset, event istructs.IWLogEvent) (err error) {
		offs := int64(wlogOffset)
		return callback(
			&key{data: map[string]interface{}{Field_Offset: offs}},
			&wLogValue{
				event:  event,
				offset: offs,
			})
	}
	return s.eventsFunc().ReadWLog(s.ctx, k.wsid, k.offset, k.count, cb)
}
