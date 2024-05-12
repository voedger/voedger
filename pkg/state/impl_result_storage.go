/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package state

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

type resultStorage struct {
	cmdResultBuilderFunc CmdResultBuilderFunc
	appStructsFunc       AppStructsFunc
	qryResultType        QNameFunc
	qryCallback          ExecQueryCallbackFunc
	qryValueBuilder      *resultValueBuilder // last value builder
}

func newCmdResultStorage(cmdResultBuilderFunc CmdResultBuilderFunc) *resultStorage {
	return &resultStorage{
		cmdResultBuilderFunc: cmdResultBuilderFunc,
	}
}

func newQueryResultStorage(appStructsFunc AppStructsFunc, qryResultType QNameFunc, qryCallback ExecQueryCallbackFunc) *resultStorage {
	return &resultStorage{
		appStructsFunc: appStructsFunc,
		qryResultType:  qryResultType,
		qryCallback:    qryCallback,
	}
}

func (s *resultStorage) NewKeyBuilder(_ appdef.QName, _ istructs.IStateKeyBuilder) istructs.IStateKeyBuilder {
	return newResultKeyBuilder()
}

func (s *resultStorage) Validate([]ApplyBatchItem) (err error) {
	panic("not applicable")
}

func (s *resultStorage) sendPrevQueryObject() error {
	if s.qryCallback != nil && s.qryValueBuilder != nil { // query processor, there's unsent object
		obj, err := s.qryValueBuilder.resultBuilder.Build()
		if err != nil {
			return err
		}
		s.qryValueBuilder = nil
		return s.qryCallback()(obj)
	}
	return nil
}

func (s *resultStorage) ApplyBatch([]ApplyBatchItem) (err error) {
	return s.sendPrevQueryObject()
}

func (s *resultStorage) ProvideValueBuilder(istructs.IStateKeyBuilder, istructs.IStateValueBuilder) (istructs.IStateValueBuilder, error) {
	if s.qryCallback != nil { // query processor
		s.sendPrevQueryObject()
		builder := s.appStructsFunc().ObjectBuilder(s.qryResultType())
		s.qryValueBuilder = &resultValueBuilder{resultBuilder: builder}
		return s.qryValueBuilder, nil
	}
	// command processor
	builder := s.cmdResultBuilderFunc()
	return &resultValueBuilder{resultBuilder: builder}, nil
}
