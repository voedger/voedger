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
	rw                   istructs.IRowWriter
	cmdResultBuilderFunc CmdResultBuilderFunc
	//result               map[string]interface{}
}

func (s *cmdResultStorage) NewKeyBuilder(entity appdef.QName, _ istructs.IStateKeyBuilder) istructs.IStateKeyBuilder {
	return newCmdResultKeyBuilder(entity)
}

func (s *cmdResultStorage) Validate([]ApplyBatchItem) (err error) {
	return nil
}

func (s *cmdResultStorage) ApplyBatch([]ApplyBatchItem) (err error) {
	return nil
}

func (s *cmdResultStorage) ProvideValueBuilder(istructs.IStateKeyBuilder, istructs.IStateValueBuilder) istructs.IStateValueBuilder {
	return s.cmdResultBuilderFunc()
}

func (s *cmdResultStorage) ProvideValueBuilderForUpdate(istructs.IStateKeyBuilder, istructs.IStateValue, istructs.IStateValueBuilder) istructs.IStateValueBuilder {
	return nil
}

func GetCmdResultBuilderFunc() CmdResultBuilderFunc {
	cmdResBuilder := newCmdResultBuilder()
	return func() istructs.IStateValueBuilder {
		return cmdResBuilder
	}
}

// value
type cmdResult struct {
	baseStateValue
	value map[string]interface{}
}

func (c *cmdResult) AsInt32(name string) int32     { return c.value[name].(int32) }
func (c *cmdResult) AsInt64(name string) int64     { return c.value[name].(int64) }
func (c *cmdResult) AsFloat32(name string) float32 { return c.value[name].(float32) }
func (c *cmdResult) AsFloat64(name string) float64 { return c.value[name].(float64) }
func (c *cmdResult) AsBytes(name string) []byte    { return c.value[name].([]byte) }
func (c *cmdResult) AsString(name string) string   { return c.value[name].(string) }
func (c *cmdResult) ToJSON(opts ...interface{}) (string, error) {
	bb, err := json.Marshal(c.value)
	if err != nil {
		return "", err
	}
	return string(bb), nil
}

func (c *cmdResult) AsValue(name string) istructs.IStateValue { return c }

func (c *cmdResult) AsRecord(name string) (record istructs.IRecord)  { panic(errNotImplemented) }
func (c *cmdResult) AsEvent(name string) (event istructs.IDbEvent)   { panic(errNotImplemented) }
func (c *cmdResult) Elements(_ string, _ func(el istructs.IElement)) { panic(errNotImplemented) }
func (c *cmdResult) Containers(cb func(container string))            { panic(errNotImplemented) }
func (c *cmdResult) QName() appdef.QName                             { panic(errNotImplemented) }
func (c *cmdResult) Length() int                                     { panic(errNotImplemented) }
func (c *cmdResult) GetAsString(index int) string                    { panic(errNotImplemented) }
func (c *cmdResult) GetAsBytes(index int) []byte                     { panic(errNotImplemented) }
func (c *cmdResult) GetAsInt32(index int) int32                      { panic(errNotImplemented) }
func (c *cmdResult) GetAsInt64(index int) int64                      { panic(errNotImplemented) }
func (c *cmdResult) GetAsFloat32(index int) float32                  { panic(errNotImplemented) }
func (c *cmdResult) GetAsFloat64(index int) float64                  { panic(errNotImplemented) }
func (c *cmdResult) GetAsQName(index int) appdef.QName               { panic(errNotImplemented) }
func (c *cmdResult) GetAsBool(index int) bool                        { panic(errNotImplemented) }
func (c *cmdResult) GetAsValue(index int) istructs.IStateValue       { panic(errNotImplemented) }

// builder
type cmdResultBuilder struct {
	result *cmdResult
}

func (c *cmdResultBuilder) BuildValue() istructs.IStateValue {
	return c.result
}

func (c *cmdResultBuilder) ElementBuilder(containerName string) istructs.IElementBuilder {
	return c
}

func (c *cmdResultBuilder) PutInt32(name string, value int32) {
	c.result.value[name] = value
	//c.cmdResultBuilderFunc.PutInt32(name, value)
}

func (c *cmdResultBuilder) PutInt64(name string, value int64) {
	c.result.value[name] = value
	//c.cmdResultBuilderFunc.PutInt64(name, value)
}
func (c *cmdResultBuilder) PutBytes(name string, value []byte) {
	c.result.value[name] = value
	//c.cmdResultBuilderFunc.PutBytes(name, value)
}
func (c *cmdResultBuilder) PutString(name, value string) {
	c.result.value[name] = value
	//c.cmdResultBuilderFunc.PutString(name, value)
}
func (c *cmdResultBuilder) PutBool(name string, value bool) {
	c.result.value[name] = value
	//c.cmdResultBuilderFunc.PutBool(name, value)
}
func (c *cmdResultBuilder) PutChars(name string, value string) {
	c.result.value[name] = value
	//c.cmdResultBuilderFunc.PutChars(name, value)
}
func (c *cmdResultBuilder) PutFloat32(name string, value float32) {
	c.result.value[name] = value
	//c.cmdResultBuilderFunc.PutFloat32(name, value)
}
func (c *cmdResultBuilder) PutFloat64(name string, value float64) {
	c.result.value[name] = value
	//c.cmdResultBuilderFunc.PutFloat64(name, value)
}
func (c *cmdResultBuilder) PutQName(name string, value appdef.QName) {
	c.result.value[name] = value
	//c.cmdResultBuilderFunc.PutQName(name, value)
}
func (c *cmdResultBuilder) PutNumber(name string, value float64) {
	c.result.value[name] = value
	//c.cmdResultBuilderFunc.PutNumber(name, value)
}
func (c *cmdResultBuilder) PutRecordID(name string, value istructs.RecordID) {
	c.result.value[name] = value
	//c.cmdResultBuilderFunc.PutRecordID(name, value)
}

func (c *cmdResultBuilder) ToJSON(opts ...interface{}) (string, error) {
	return c.result.ToJSON()
}

func (c *cmdResultBuilder) PutRecord(name string, record istructs.IRecord) { panic(errNotImplemented) }
func (c *cmdResultBuilder) PutEvent(name string, event istructs.IDbEvent)  { panic(errNotImplemented) }
func (c *cmdResultBuilder) Build() istructs.IValue                         { panic(errNotImplemented) }

func newCmdResultBuilder() istructs.IStateValueBuilder {
	return &cmdResultBuilder{
		result: &cmdResult{
			value: make(map[string]interface{}),
		},
	}
}
