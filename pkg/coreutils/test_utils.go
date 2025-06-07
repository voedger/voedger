/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package coreutils

import (
	"encoding/json"
	"maps"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

const TooBigNumberStr = "1111111111111111111111111111111111999999999999999999999999999111111111111111111111111111111111111111119999999999999999999999999991111111111111111111111111111111111111111199999999999999999999999999911111111111111111111111111111111111111111999999999999999999999999999111111111111111111111111111111111111111119999999999999999999999999991111111"

// ICUDRow, IObject
type TestObject struct {
	istructs.NullObject
	Name        appdef.QName
	ID_         istructs.RecordID
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

func (o *TestObject) PutInt8(name string, value int8)                  { o.Data[name] = value }
func (o *TestObject) PutInt16(name string, value int16)                { o.Data[name] = value }
func (o *TestObject) PutInt32(name string, value int32)                { o.Data[name] = value }
func (o *TestObject) PutInt64(name string, value int64)                { o.Data[name] = value }
func (o *TestObject) PutFloat32(name string, value float32)            { o.Data[name] = value }
func (o *TestObject) PutFloat64(name string, value float64)            { o.Data[name] = value }
func (o *TestObject) PutBytes(name string, value []byte)               { o.Data[name] = value }
func (o *TestObject) PutString(name, value string)                     { o.Data[name] = value }
func (o *TestObject) PutQName(name string, value appdef.QName)         { o.Data[name] = value }
func (o *TestObject) PutBool(name string, value bool)                  { o.Data[name] = value }
func (o *TestObject) PutRecordID(name string, value istructs.RecordID) { o.Data[name] = value }
func (o *TestObject) PutNumber(name string, value json.Number)         { o.Data[name] = value }
func (o *TestObject) PutChars(name string, value string)               { o.Data[name] = value }
func (o *TestObject) PutFromJSON(value map[string]any)                 { maps.Copy(o.Data, value) }

func (o *TestObject) ID() istructs.RecordID {
	if o.ID_ == istructs.NullRecordID {
		return o.Data[appdef.SystemField_ID].(istructs.RecordID)
	}
	return o.ID_
}
func (o *TestObject) QName() appdef.QName       { return o.Name }
func (o *TestObject) Parent() istructs.RecordID { return o.Parent_ }

func (o *TestObject) IsActivated() bool {
	if !o.IsNew_ {
		if d, ok := o.Data[appdef.SystemField_IsActive]; ok {
			if active, ok := d.(bool); ok {
				return active
			}
		}
	}
	return false
}

func (o *TestObject) IsDeactivated() bool {
	if !o.IsNew_ {
		if d, ok := o.Data[appdef.SystemField_IsActive]; ok {
			if active, ok := d.(bool); ok {
				return !active
			}
		}
	}
	return false
}

func (o *TestObject) IsNew() bool { return o.IsNew_ }

func (o *TestObject) SpecifiedValues(cb func(appdef.IField, any) bool) {
	if !cb(&MockIField{FieldName: appdef.SystemField_ID, FieldDataKind: appdef.DataKind_RecordID}, o.ID_) {
		return
	}
	if !cb(&MockIField{FieldName: appdef.SystemField_IsActive, FieldDataKind: appdef.DataKind_bool}, true) {
		return
	}
	if !cb(&MockIField{FieldName: appdef.SystemField_QName, FieldDataKind: appdef.DataKind_QName}, o.Name) {
		return
	}
	for name, value := range o.Data {
		if name == appdef.SystemField_ID || name == appdef.SystemField_QName || name == appdef.SystemField_IsActive {
			continue
		}
		if !cb(&MockIField{FieldName: name, FieldDataKind: intfToDataKind(value)}, value) {
			break
		}
	}
}
func (o *TestObject) AsRecord() istructs.IRecord                 { return o }
func (o *TestObject) AsEvent(appdef.FieldName) istructs.IDbEvent { panic("not implemented") }

func (o *TestObject) AsInt8(name string) int8 {
	if resIntf, ok := o.Data[name]; ok {
		return resIntf.(int8)
	}
	return 0
}

func (o *TestObject) AsInt16(name string) int16 {
	if resIntf, ok := o.Data[name]; ok {
		return resIntf.(int16)
	}
	return 0
}

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
	if name == appdef.SystemField_ID {
		return o.ID()
	}
	if resIntf, ok := o.Data[name]; ok {
		return resIntf.(istructs.RecordID)
	}
	return istructs.NullRecordID
}
func (o *TestObject) Children(container ...string) func(func(istructs.IObject) bool) {
	cc := make(map[string]bool)
	for _, c := range container {
		cc[c] = true
	}
	return func(cb func(istructs.IObject) bool) {
		for c, children := range o.Containers_ {
			if len(cc) == 0 || cc[c] {
				for _, child := range children {
					if !cb(child) {
						break
					}
				}
			}
		}
	}
}

func intfToDataKind(value interface{}) appdef.DataKind {
	switch value.(type) {
	case string:
		return appdef.DataKind_string
	case int8: // #3434 : small integers
		return appdef.DataKind_int8
	case int16: // #3434 : small integers
		return appdef.DataKind_int16
	case int32:
		return appdef.DataKind_int32
	case int64:
		return appdef.DataKind_int64
	case float32:
		return appdef.DataKind_float32
	case float64:
		return appdef.DataKind_float64
	case bool:
		return appdef.DataKind_bool
	case appdef.QName:
		return appdef.DataKind_QName
	case map[string]interface{}:
		return appdef.DataKind_Record
	default:
		return appdef.DataKind_null
	}
}

type MockIField struct {
	appdef.IField
	FieldName     string
	FieldDataKind appdef.DataKind
}

const notImplemented = "not implemented"

func (f *MockIField) Comment() string                               { panic(notImplemented) }
func (f *MockIField) CommentLines() []string                        { panic(notImplemented) }
func (f *MockIField) Name() appdef.FieldName                        { return f.FieldName }
func (f *MockIField) Data() appdef.IData                            { panic(notImplemented) }
func (f *MockIField) DataKind() appdef.DataKind                     { return f.FieldDataKind }
func (f *MockIField) Required() bool                                { return false }
func (f *MockIField) Verifiable() bool                              { panic(notImplemented) }
func (f *MockIField) VerificationKind(appdef.VerificationKind) bool { panic(notImplemented) }
func (f *MockIField) IsFixedWidth() bool                            { panic(notImplemented) }
func (f *MockIField) IsSys() bool                                   { panic(notImplemented) }
func (f *MockIField) Constraints() map[appdef.ConstraintKind]appdef.IConstraint {
	panic(notImplemented)
}

func (o *TestObject) Fields(cb func(iField appdef.IField) bool) {
	for name := range o.Data {
		if !cb(&MockIField{FieldName: name}) {
			break
		}
	}
}
func (o *TestObject) Containers(cb func(string) bool) {
	for containerName := range o.Containers_ {
		if !cb(containerName) {
			break
		}
	}
}
