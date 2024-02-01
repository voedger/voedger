/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"context"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

type pLogStorage struct {
	ctx             context.Context
	eventsFunc      eventsFunc
	partitionIDFunc PartitionIDFunc
}

func (s *pLogStorage) NewKeyBuilder(appdef.QName, istructs.IStateKeyBuilder) istructs.IStateKeyBuilder {
	return &pLogKeyBuilder{
		logKeyBuilder: logKeyBuilder{
			offset: istructs.FirstOffset,
			count:  1,
		},
		partitionID: s.partitionIDFunc(),
	}
}
func (s *pLogStorage) Get(key istructs.IStateKeyBuilder) (value istructs.IStateValue, err error) {
	err = s.Read(key, func(_ istructs.IKey, v istructs.IStateValue) (err error) {
		value = v
		return nil
	})
	return value, err
}
func (s *pLogStorage) Read(kb istructs.IStateKeyBuilder, callback istructs.ValueCallback) (err error) {
	k := kb.(*pLogKeyBuilder)
	cb := func(plogOffset istructs.Offset, event istructs.IPLogEvent) (err error) {
		offs := int64(plogOffset)
		return callback(
			&key{data: map[string]interface{}{Field_Offset: offs}},
			&pLogValue{
				event:  event,
				offset: offs,
			})
	}
	return s.eventsFunc().ReadPLog(s.ctx, k.partitionID, k.offset, k.count, cb)
}
