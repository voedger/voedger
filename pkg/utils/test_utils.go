/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package coreutils

import (
	"fmt"
	"maps"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	ibus "github.com/voedger/voedger/staging/src/github.com/untillpro/airs-ibus"
)

var (
	TestNow      = time.Now()
	TestTimeFunc = TimeFunc(func() time.Time { return TestNow })
)

// ICUDRow, IObject, IFields
type TestObject struct {
	istructs.NullObject
	Name        appdef.QName
	Id          istructs.RecordID
	Parent_     istructs.RecordID
	Data        map[string]interface{}
	Containers_ map[string][]*TestObject
	IsNew_      bool
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
func (o *TestObject) PutFromJSON(value map[string]any)                 { maps.Copy(o.Data, value) }

func (o *TestObject) ID() istructs.RecordID     { return o.Id }
func (o *TestObject) QName() appdef.QName       { return o.Name }
func (o *TestObject) Parent() istructs.RecordID { return o.Parent_ }
func (o *TestObject) IsNew() bool               { return o.IsNew_ }
func (o *TestObject) ModifiedFields(cb func(string, interface{})) {
	for name, value := range o.Data {
		cb(name, value)
	}
}
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
func (o *TestObject) Children(container string, cb func(istructs.IObject)) {
	iterate := func(cc []*TestObject) {
		for _, c := range cc {
			cb(c)
		}
	}

	if container == "" {
		for _, cc := range o.Containers_ {
			iterate(cc)
		}
	} else {
		if cc, ok := o.Containers_[container]; ok {
			iterate(cc)
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
func (o *TestObject) Fields() []appdef.IField {
	res := []appdef.IField{}
	for name, val := range o.Data {
		res = append(res, &TestIField{
			name: name,
			val:  val,
		})
	}
	return res
}
func (o *TestObject) Field(name string) appdef.IField {
	val, ok := o.Data[name]
	if !ok {
		return nil
	}
	return &TestIField{
		name: name,
		val:  val,
	}
}

func (o *TestObject) FieldCount() int {
	return len(o.Data)
}
func (o *TestObject) RefField(name string) appdef.IRefField {
	panic("not implemented")
}

func (o *TestObject) RefFields() []appdef.IRefField {
	panic("not implemented")
}

func (o *TestObject) UserFieldCount() int {
	panic("not implemented")
}

type TestIField struct {
	name string
	val  interface{}
}

func (f *TestIField) Comment() string        { return "" }
func (f *TestIField) CommentLines() []string { return nil }
func (f *TestIField) Name() string           { return f.name }
func (f *TestIField) Data() appdef.IData     { return nil }
func (f *TestIField) DataKind() appdef.DataKind {
	switch f.val.(type) {
	case int32:
		return appdef.DataKind_int32
	case int64:
		return appdef.DataKind_int64
	case string:
		return appdef.DataKind_string
	case []byte:
		return appdef.DataKind_bytes
	case bool:
		return appdef.DataKind_bool
	case float32:
		return appdef.DataKind_float32
	case float64:
		return appdef.DataKind_float64
	case appdef.QName:
		return appdef.DataKind_QName
	case istructs.RecordID:
		return appdef.DataKind_RecordID
	default:
		panic(fmt.Sprint("unsupported data type:", f.val))
	}
}
func (f *TestIField) Required() bool                                { return false }
func (f *TestIField) Verifiable() bool                              { return false }
func (f *TestIField) VerificationKind(appdef.VerificationKind) bool { return false }
func (f *TestIField) IsFixedWidth() bool                            { return true }
func (f *TestIField) IsSys() bool                                   { return false }
func (f *TestIField) Constraints() map[appdef.ConstraintKind]appdef.IConstraint {
	return map[appdef.ConstraintKind]appdef.IConstraint{}
}

func GetTestBustTimeout() time.Duration {
	if IsDebug() {
		return time.Hour
	}
	return ibus.DefaultTimeout
}
