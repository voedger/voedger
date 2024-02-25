/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package state

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

type argStorage struct {
	argFunc ArgFunc
}

func (s *argStorage) NewKeyBuilder(_ appdef.QName, _ istructs.IStateKeyBuilder) istructs.IStateKeyBuilder {
	return newKeyBuilder(Event, appdef.NullQName)
}
func (s *argStorage) Get(_ istructs.IStateKeyBuilder) (istructs.IStateValue, error) {
	return &objectValue{
		object: s.argFunc(),
	}, nil
}
