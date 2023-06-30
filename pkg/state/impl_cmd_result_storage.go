/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package state

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

type cmdResultStorage struct {
	istructs.IStateValueBuilder
	cmdResultBuilderFunc CmdResultBuilderFunc
}

func (s *cmdResultStorage) NewKeyBuilder(entity appdef.QName, _ istructs.IStateKeyBuilder) istructs.IStateKeyBuilder {
	return newCmdResultKeyBuilder(entity)
}

func (s *cmdResultStorage) Validate([]ApplyBatchItem) (err error) {
	panic("not applicable")
}

func (s *cmdResultStorage) ApplyBatch([]ApplyBatchItem) (err error) {
	panic("not applicable")
}

func (s *cmdResultStorage) ProvideValueBuilder(istructs.IStateKeyBuilder, istructs.IStateValueBuilder) istructs.IStateValueBuilder {
	return &cmdResultValueBuilder{cmdResultBuilder: s.cmdResultBuilderFunc()}
}
