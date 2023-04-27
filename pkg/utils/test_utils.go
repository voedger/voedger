/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package coreutils

import (
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

var (
	TestNow      = time.Now()
	TestTimeFunc = func() time.Time { return TestNow }
)

type TestObject struct {
	istructs.NullObject
	Name        appdef.QName
	Id          istructs.RecordID
	Parent_     istructs.RecordID
	Data        map[string]interface{}
	Containers_ map[string][]*TestObject
}

type TestSchema struct {
	Fields_     map[string]appdef.DataKind
	Containers_ map[string]appdef.QName
	QName_      appdef.QName
	Signleton_  bool
}

type TestSchemas struct {
	Schemas_ map[appdef.QName]appdef.Schema
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

func (s TestSchemas) Schemas(cb func(appdef.Schema)) {
	for _, s := range s.Schemas_ {
		cb(s)
	}
}
func (s TestSchemas) SchemaCount() int                             { return len(s.Schemas_) }
func (s TestSchemas) SchemaByName(name appdef.QName) appdef.Schema { return s.Schemas_[name] }
func (s TestSchemas) Schema(name appdef.QName) appdef.Schema {
	if schema := s.SchemaByName(name); schema != nil {
		return schema
	}
	return nil
}

func (o *TestObject) PutInt32(name string, value int32)                { o.Data[name] = value }
func (o *TestObject) PutInt64(name string, value int64)                { o.Data[name] = value }
func (o *TestObject) PutFloat32(name string, value float32)            { o.Data[name] = value }
func (o *TestObject) PutFloat64(name string, value float64)            { o.Data[name] = value }
func (o *TestObject) PutBytes(name string, value []byte)               { o.Data[name] = value }
func (o *TestObject) PutString(name, value string)                     { o.Data[name] = value }
func (o *TestObject) PutQName(name string, value appdef.QName)         { o.Data[name] = value }
func (o *TestObject) PutBool(name string, value bool)                  { o.Data[name] = value }
func (o *TestObject) PutRecordID(name string, value istructs.RecordID) { o.Data[name] = value }
func (o *TestObject) PutNumber(name string, value float64)             { o.Data[name] = value }
func (o *TestObject) PutChars(name string, value string)               { o.Data[name] = value }

func (o *TestObject) ID() istructs.RecordID      { return o.Id }
func (o *TestObject) QName() appdef.QName        { return o.Name }
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
func (o *TestObject) AsQName(name string) appdef.QName {
	qNameIntf, ok := o.Data[name]
	if !ok {
		return appdef.NullQName
	}
	return qNameIntf.(appdef.QName)
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

func (s TestSchema) Cache() appdef.SchemaCache { panic("implement me") }
func (s TestSchema) QName() appdef.QName       { return s.QName_ }
func (s TestSchema) Kind() appdef.SchemaKind   { return appdef.SchemaKind_FakeLast }
func (s TestSchema) Field(name string) appdef.Field {
	if k, ok := s.Fields_[name]; ok {
		fld := feildDescr{name: name, kind: k}
		return &fld
	}
	return nil
}
func (s TestSchema) FieldCount() int { return len(s.Fields_) }
func (s TestSchema) Fields(cb func(appdef.Field)) {
	for n, k := range s.Fields_ {
		fld := feildDescr{name: n, kind: k}
		cb(&fld)
	}
}
func (s TestSchema) Container(name string) appdef.Container {
	if s, ok := s.Containers_[name]; ok {
		cont := contDescr{name: name, schema: s}
		return &cont
	}
	return nil
}
func (s TestSchema) ContainerCount() int { return len(s.Containers_) }
func (s TestSchema) Containers(cb func(appdef.Container)) {
	for n, s := range s.Containers_ {
		cont := contDescr{name: n, schema: s}
		cb(&cont)
	}
}
func (s TestSchema) ContainerSchema(name string) appdef.Schema { panic("implement me") }
func (s TestSchema) Singleton() bool                           { return s.Signleton_ }
func (s TestSchema) Validate() error                           { return nil }

type feildDescr struct {
	name string
	kind appdef.DataKind
}

func (f feildDescr) Name() string                                  { return f.name }
func (f feildDescr) DataKind() appdef.DataKind                     { return f.kind }
func (f feildDescr) Required() bool                                { return false }
func (f feildDescr) Verifiable() bool                              { return false }
func (f feildDescr) VerificationKind(appdef.VerificationKind) bool { return false }
func (f feildDescr) IsFixedWidth() bool                            { return f.kind.IsFixed() }
func (f feildDescr) IsSys() bool                                   { return appdef.IsSysField(f.name) }

type contDescr struct {
	name   string
	schema appdef.QName
}

func (c contDescr) Name() string             { return c.name }
func (c contDescr) Schema() appdef.QName     { return c.schema }
func (c contDescr) MinOccurs() appdef.Occurs { return 0 }
func (c contDescr) MaxOccurs() appdef.Occurs { return appdef.Occurs_Unbounded }
func (c contDescr) IsSys() bool              { return appdef.IsSysContainer(c.name) }
