/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package state

import (
	"encoding/json"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

type cmdResultStorage struct {
	*keyBuilder
	istructs.IStateValueBuilder
	rw               istructs.IRowWriter
	cmdResultBuilder istructs.IObjectBuilder
}

func (s *cmdResultStorage) NewKeyBuilder(appdef.QName, istructs.IStateKeyBuilder) istructs.IStateKeyBuilder {
	return newCmdResultKeyBuilder()
}

func (s *cmdResultStorage) Validate([]ApplyBatchItem) (err error) {
	return nil
}

func (s *cmdResultStorage) ApplyBatch([]ApplyBatchItem) (err error) {
	return nil
}

func (s *cmdResultStorage) Read(istructs.IStateKeyBuilder, istructs.ValueCallback) (err error) {
	return nil
}

func (s *cmdResultStorage) ProvideValueBuilder(istructs.IStateKeyBuilder, istructs.IStateValueBuilder) istructs.IStateValueBuilder {
	return nil
}

func (s *cmdResultStorage) ProvideValueBuilderForUpdate(istructs.IStateKeyBuilder, istructs.IStateValue, istructs.IStateValueBuilder) istructs.IStateValueBuilder {
	return nil
}

func (s *cmdResultStorage) toJSON(sv istructs.IStateValue, _ ...interface{}) (string, error) {
	bb, err := json.Marshal(sv.(*cmdResultStorageValue).result)
	return string(bb), err
}
