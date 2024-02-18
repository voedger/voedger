/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package state

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

type eventStorage struct {
	eventFunc PLogEventFunc
}

func (s *eventStorage) NewKeyBuilder(_ appdef.QName, _ istructs.IStateKeyBuilder) istructs.IStateKeyBuilder {
	return newKeyBuilder(Event, appdef.NullQName)
}
func (s *eventStorage) Get(_ istructs.IStateKeyBuilder) (istructs.IStateValue, error) {
	return &pLogValue{
		event: s.eventFunc(),
	}, nil
}
