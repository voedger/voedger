/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package storages

import (
	"encoding/json"
	"reflect"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys"
)

type resultStorage struct {
	resultBuilderFunc state.ObjectBuilderFunc
}

type resultKeyBuilder struct {
	baseKeyBuilder
}

func (b *resultKeyBuilder) Equals(src istructs.IKeyBuilder) bool {
	_, ok := src.(*resultKeyBuilder)
	return ok
}

func NewResultStorage(resultBuilderFunc state.ObjectBuilderFunc) state.IStateStorage {
	return &resultStorage{
		resultBuilderFunc: resultBuilderFunc,
	}
}

func (s *resultStorage) NewKeyBuilder(_ appdef.QName, _ istructs.IStateKeyBuilder) istructs.IStateKeyBuilder {
	return &resultKeyBuilder{
		baseKeyBuilder: baseKeyBuilder{storage: sys.Storage_Result},
	}
}

func (s *resultStorage) Validate([]state.ApplyBatchItem) (err error) {
	panic("not applicable")
}

func (s *resultStorage) ApplyBatch([]state.ApplyBatchItem) (err error) {
	return nil
}

func (s *resultStorage) ProvideValueBuilder(istructs.IStateKeyBuilder, istructs.IStateValueBuilder) (istructs.IStateValueBuilder, error) {
	return &resultValueBuilder{resultBuilder: s.resultBuilderFunc()}, nil
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
	return &ObjectStateValue{object: o}
}

func (c *resultValueBuilder) PutInt8(name string, value int8) {
	c.resultBuilder.PutInt8(name, value)
}

func (c *resultValueBuilder) PutInt16(name string, value int16) {
	c.resultBuilder.PutInt16(name, value)
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
func (c *resultValueBuilder) PutNumber(name string, value json.Number) {
	c.resultBuilder.PutNumber(name, value)
}
func (c *resultValueBuilder) PutRecordID(name string, value istructs.RecordID) {
	c.resultBuilder.PutRecordID(name, value)
}
