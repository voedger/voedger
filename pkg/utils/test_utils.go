/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package coreutils

import (
	"time"

	"github.com/voedger/voedger/pkg/istructs"
)

var (
	TestNow      = time.Now()
	TestTimeFunc = func() time.Time { return TestNow }
)

type TestObject struct {
	istructs.NullObject
	Name        istructs.QName
	Id          istructs.RecordID
	Parent_     istructs.RecordID
	Data        map[string]interface{}
	Containers_ map[string][]*TestObject
}

type TestSchema struct {
	Fields_     map[string]istructs.DataKindType
	Containers_ map[string]istructs.QName
	QName_      istructs.QName
}

type TestSchemas struct {
	Schemas_ map[istructs.QName]istructs.ISchema
}

type TestValue struct {
	*TestObject
}

func (v *TestValue) AsRecord(name string) (record istructs.IRecord) {
	return v.Data[name].(istructs.IRecord)
}
func (v *TestValue) AsEvent(name string) (event istructs.IDbEvent) {
	return v.Data[name].(istructs.IDbEvent)
}

func (s TestSchemas) Schema(schema istructs.QName) istructs.ISchema { return s.Schemas_[schema] }
func (s TestSchemas) Schemas(cb func(istructs.QName)) {
	for n := range s.Schemas_ {
		cb(n)
	}
}

func (o *TestObject) PutInt32(name string, value int32)                { o.Data[name] = value }
func (o *TestObject) PutInt64(name string, value int64)                { o.Data[name] = value }
func (o *TestObject) PutFloat32(name string, value float32)            { o.Data[name] = value }
func (o *TestObject) PutFloat64(name string, value float64)            { o.Data[name] = value }
func (o *TestObject) PutBytes(name string, value []byte)               { o.Data[name] = value }
func (o *TestObject) PutString(name, value string)                     { o.Data[name] = value }
func (o *TestObject) PutQName(name string, value istructs.QName)       { o.Data[name] = value }
func (o *TestObject) PutBool(name string, value bool)                  { o.Data[name] = value }
func (o *TestObject) PutRecordID(name string, value istructs.RecordID) { o.Data[name] = value }
func (o *TestObject) PutNumber(name string, value float64)             { o.Data[name] = value }
func (o *TestObject) PutChars(name string, value string)               { o.Data[name] = value }

func (o *TestObject) ID() istructs.RecordID      { return o.Id }
func (o *TestObject) QName() istructs.QName      { return o.Name }
func (o *TestObject) Parent() istructs.RecordID  { return o.Parent_ }
func (o *TestObject) AsRecord() istructs.IRecord { return o }
func (o *TestObject) AsInt32(name string) int32 {
	if resIntf, ok := o.Data[name]; ok {
		return resIntf.(int32)
	}
	return 0
}
func (o *TestObject) AsInt64(name string) int64 {
	if resIntf, ok := o.Data[name]; ok {
		return resIntf.(int64)
	}
	return 0
}
func (o *TestObject) AsFloat32(name string) float32 {
	if resIntf, ok := o.Data[name]; ok {
		return resIntf.(float32)
	}
	return 0
}
func (o *TestObject) AsFloat64(name string) float64 {
	if resIntf, ok := o.Data[name]; ok {
		return resIntf.(float64)
	}
	return 0
}
func (o *TestObject) AsBytes(name string) []byte {
	if resIntf, ok := o.Data[name]; ok {
		return resIntf.([]byte)
	}
	return nil
}
func (o *TestObject) AsString(name string) string {
	if resIntf, ok := o.Data[name]; ok {
		return resIntf.(string)
	}
	return ""
}
func (o *TestObject) AsBool(name string) bool {
	if resIntf, ok := o.Data[name]; ok {
		return resIntf.(bool)
	}
	return false
}
func (o *TestObject) AsQName(name string) istructs.QName {
	qNameIntf, ok := o.Data[name]
	if !ok {
		return istructs.NullQName
	}
	return qNameIntf.(istructs.QName)
}
func (o *TestObject) AsRecordID(name string) istructs.RecordID {
	if resIntf, ok := o.Data[name]; ok {
		return resIntf.(istructs.RecordID)
	}
	return istructs.NullRecordID
}
func (o *TestObject) Elements(container string, cb func(el istructs.IElement)) {
	if objects, ok := o.Containers_[container]; ok {
		for _, object := range objects {
			cb(object)
		}
	}
}
func (o *TestObject) FieldNames(cb func(name string)) {
	for name := range o.Data {
		cb(name)
	}
}
func (o *TestObject) Containers(cb func(container string)) {
	for containerName := range o.Containers_ {
		cb(containerName)
	}
}

func (s TestSchema) QName() istructs.QName         { return s.QName_ }
func (s TestSchema) Kind() istructs.SchemaKindType { return istructs.SchemaKind_FakeLast }
func (s TestSchema) Fields(cb func(fieldName string, kind istructs.DataKindType)) {
	for k, v := range s.Fields_ {
		cb(k, v)
	}
}
func (s TestSchema) ForEachField(cb func(istructs.IFieldDescr)) {
	for n, k := range s.Fields_ {
		cb(&feildDescr{name: n, kind: k})
	}
}
func (s TestSchema) Containers(cb func(containerName string, schema istructs.QName)) {
	for name, schema := range s.Containers_ {
		cb(name, schema)
	}
}
func (s TestSchema) ForEachContainer(cb func(istructs.IContainerDescr)) {
	panic("implement me")
}

type feildDescr struct {
	name string
	kind istructs.DataKindType
}

func (f feildDescr) Name() string                    { return f.name }
func (f feildDescr) DataKind() istructs.DataKindType { return f.kind }
func (f feildDescr) Required() bool                  { return false }
func (f feildDescr) Verifiable() bool                { return false }
