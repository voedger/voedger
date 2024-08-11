/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package state

import (
	"reflect"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys"
)

type resultStorage struct {
	cmdResultBuilderFunc ObjectBuilderFunc
	appStructsFunc       AppStructsFunc
	resultBuilderFunc    ObjectBuilderFunc
	qryCallback          ExecQueryCallbackFunc
	qryValueBuilder      *resultValueBuilder // last value builder
}

type resultKeyBuilder struct {
	baseKeyBuilder
}

func (b *resultKeyBuilder) Storage() appdef.QName {
	return sys.Storage_Result
}

func (b *resultKeyBuilder) Equals(src istructs.IKeyBuilder) bool {
	_, ok := src.(*resultKeyBuilder)
	return ok
}

func newCmdResultStorage(cmdResultBuilderFunc ObjectBuilderFunc) *resultStorage {
	return &resultStorage{
		cmdResultBuilderFunc: cmdResultBuilderFunc,
	}
}

func newQueryResultStorage(appStructsFunc AppStructsFunc, resultBuilderFunc ObjectBuilderFunc, qryCallback ExecQueryCallbackFunc) *resultStorage {
	return &resultStorage{
		appStructsFunc:    appStructsFunc,
		resultBuilderFunc: resultBuilderFunc,
		qryCallback:       qryCallback,
	}
}

func (s *resultStorage) NewKeyBuilder(_ appdef.QName, _ istructs.IStateKeyBuilder) istructs.IStateKeyBuilder {
	return &resultKeyBuilder{}
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
		err := s.sendPrevQueryObject()
		if err != nil {
			return nil, err
		}
		s.qryValueBuilder = &resultValueBuilder{resultBuilder: s.resultBuilderFunc()}
		return s.qryValueBuilder, nil
	}
	// command processor
	builder := s.cmdResultBuilderFunc()
	return &resultValueBuilder{resultBuilder: builder}, nil
}

type resultValueBuilder struct {
	baseValueBuilder
	resultBuilder istructs.IObjectBuilder
}

func (c *resultValueBuilder) Equal(src istructs.IStateValueBuilder) bool {
	if _, ok := src.(*resultValueBuilder); !ok {
		return false
	}
	o1, err := c.resultBuilder.Build()
	if err != nil {
		panic(err)
	}
	o2, err := src.(*resultValueBuilder).resultBuilder.Build()
	if err != nil {
		panic(err)
	}
	return reflect.DeepEqual(o1, o2)
}

func (c *resultValueBuilder) BuildValue() istructs.IStateValue {
	o, err := c.resultBuilder.Build()
	if err != nil {
		panic(err)
	}
	return &objectValue{object: o}
}

func (c *resultValueBuilder) PutInt32(name string, value int32) {
	c.resultBuilder.PutInt32(name, value)
}

func (c *resultValueBuilder) PutInt64(name string, value int64) {
	c.resultBuilder.PutInt64(name, value)
}
func (c *resultValueBuilder) PutBytes(name string, value []byte) {
	c.resultBuilder.PutBytes(name, value)
}
func (c *resultValueBuilder) PutString(name, value string) {
	c.resultBuilder.PutString(name, value)
}
func (c *resultValueBuilder) PutBool(name string, value bool) {
	c.resultBuilder.PutBool(name, value)
}
func (c *resultValueBuilder) PutChars(name string, value string) {
	c.resultBuilder.PutChars(name, value)
}
func (c *resultValueBuilder) PutFloat32(name string, value float32) {
	c.resultBuilder.PutFloat32(name, value)
}
func (c *resultValueBuilder) PutFloat64(name string, value float64) {
	c.resultBuilder.PutFloat64(name, value)
}
func (c *resultValueBuilder) PutQName(name string, value appdef.QName) {
	c.resultBuilder.PutQName(name, value)
}
func (c *resultValueBuilder) PutNumber(name string, value float64) {
	c.resultBuilder.PutNumber(name, value)
}
func (c *resultValueBuilder) PutRecordID(name string, value istructs.RecordID) {
	c.resultBuilder.PutRecordID(name, value)
}
