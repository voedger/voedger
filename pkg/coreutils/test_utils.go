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
func (o *TestObject) PutNumber(name string, value json.Number)         { o.Data[name] = value }
func (o *TestObject) PutChars(name string, value string)               { o.Data[name] = value }
func (o *TestObject) PutFromJSON(value map[string]any)                 { maps.Copy(o.Data, value) }

func (o *TestObject) ID() istructs.RecordID     { return o.Id }
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

func (o *TestObject) ModifiedFields(cb func(string, interface{}) bool) {
	for name, value := range o.Data {
		if !cb(name, value) {
			break
		}
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
func (o *TestObject) FieldNames(cb func(name string) bool) {
	for name := range o.Data {
		if !cb(name) {
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
