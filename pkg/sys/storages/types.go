/*
  - Copyright (c) 2024-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/
package storages

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys"
)

type baseKeyBuilder struct {
	istructs.IStateKeyBuilder
	entity appdef.QName
}

func (b *baseKeyBuilder) Storage() appdef.QName {
	panic(errNotImplemented)
}
func (b *baseKeyBuilder) Entity() appdef.QName {
	return b.entity
}
func (b *baseKeyBuilder) PartitionKey() istructs.IRowWriter                { panic(ErrNotSupported) } // TODO: must be eliminated, IStateKeyBuilder must be inherited from IRowWriter
func (b *baseKeyBuilder) ClusteringColumns() istructs.IRowWriter           { panic(ErrNotSupported) } // TODO: must be eliminated, IStateKeyBuilder must be inherited from IRowWriter
func (b *baseKeyBuilder) ToBytes(istructs.WSID) (pk, cc []byte, err error) { panic(ErrNotSupported) } // TODO: must be eliminated, IStateKeyBuilder must be inherited from IRowWriter
func (b *baseKeyBuilder) PutInt32(name appdef.FieldName, value int32) {
	panic(errInt32FieldUndefined(name))
}
func (b *baseKeyBuilder) PutInt64(name appdef.FieldName, value int64) {
	panic(errInt64FieldUndefined(name))
}
func (b *baseKeyBuilder) PutFloat32(name appdef.FieldName, value float32) {
	panic(errFloat32FieldUndefined(name))
}
func (b *baseKeyBuilder) PutFloat64(name appdef.FieldName, value float64) {
	panic(errFloat64FieldUndefined(name))
}

// Puts value into bytes or raw data field.
func (b *baseKeyBuilder) PutBytes(name appdef.FieldName, value []byte) {
	panic(errBytesFieldUndefined(name))
}

// Puts value into string or raw data field.
func (b *baseKeyBuilder) PutString(name appdef.FieldName, value string) {
	panic(errStringFieldUndefined(name))
}

func (b *baseKeyBuilder) PutQName(name appdef.FieldName, value appdef.QName) {
	panic(errQNameFieldUndefined(name))
}
func (b *baseKeyBuilder) PutBool(name appdef.FieldName, value bool) {
	panic(errBoolFieldUndefined(name))
}
func (b *baseKeyBuilder) PutRecordID(name appdef.FieldName, value istructs.RecordID) {
	panic(errRecordIDFieldUndefined(name))
}
func (b *baseKeyBuilder) PutNumber(name appdef.FieldName, value float64) {
	panic(errNumberFieldUndefined(name))
}
func (b *baseKeyBuilder) PutChars(name appdef.FieldName, value string) {
	panic(errCharsFieldUndefined(name))
}
func (b *baseKeyBuilder) PutFromJSON(map[appdef.FieldName]any) {
	panic(ErrNotSupported)
}
func (b *baseKeyBuilder) Equals(src istructs.IKeyBuilder) bool {
	panic(errNotImplemented)
}

type baseValueBuilder struct {
	istructs.IStateValueBuilder
}

func (b *baseValueBuilder) Equal(src istructs.IStateValueBuilder) bool {
	return false
}
func (b *baseValueBuilder) PutInt32(name string, value int32) {
	panic(errInt32FieldUndefined(name))
}
func (b *baseValueBuilder) PutInt64(name string, value int64) {
	panic(errInt64FieldUndefined(name))
}
func (b *baseValueBuilder) PutBytes(name string, value []byte) {
	panic(errBytesFieldUndefined(name))
}
func (b *baseValueBuilder) PutString(name, value string)       { panic(errStringFieldUndefined(name)) }
func (b *baseValueBuilder) PutBool(name string, value bool)    { panic(errBoolFieldUndefined(name)) }
func (b *baseValueBuilder) PutChars(name string, value string) { panic(errCharsFieldUndefined(name)) }
func (b *baseValueBuilder) PutFloat32(name string, value float32) {
	panic(errFloat32FieldUndefined(name))
}
func (b *baseValueBuilder) PutFloat64(name string, value float64) {
	panic(errFloat64FieldUndefined(name))
}
func (b *baseValueBuilder) PutQName(name string, value appdef.QName) {
	panic(errQNameFieldUndefined(name))
}
func (b *baseValueBuilder) PutNumber(name string, value float64) {
	panic(errNumberFieldUndefined(name))
}
func (b *baseValueBuilder) PutRecordID(name string, value istructs.RecordID) {
	panic(errRecordIDFieldUndefined(name))
}
func (b *baseValueBuilder) BuildValue() istructs.IStateValue {
	panic(errNotImplemented)
}

type baseStateValue struct{}

func (v *baseStateValue) AsInt32(name string) int32        { panic(errStringFieldUndefined(name)) }
func (v *baseStateValue) AsInt64(name string) int64        { panic(errInt64FieldUndefined(name)) }
func (v *baseStateValue) AsFloat32(name string) float32    { panic(errFloat32FieldUndefined(name)) }
func (v *baseStateValue) AsFloat64(name string) float64    { panic(errFloat64FieldUndefined(name)) }
func (v *baseStateValue) AsBytes(name string) []byte       { panic(errBytesFieldUndefined(name)) }
func (v *baseStateValue) AsString(name string) string      { panic(errStringFieldUndefined(name)) }
func (v *baseStateValue) AsQName(name string) appdef.QName { panic(errQNameFieldUndefined(name)) }
func (v *baseStateValue) AsBool(name string) bool          { panic(errBoolFieldUndefined(name)) }
func (v *baseStateValue) AsValue(name string) istructs.IStateValue {
	panic(errValueFieldUndefined(name))
}
func (v *baseStateValue) AsRecordID(name string) istructs.RecordID {
	panic(errRecordIDFieldUndefined(name))
}
func (v *baseStateValue) AsRecord(name string) istructs.IRecord           { panic(errNotImplemented) }
func (v *baseStateValue) AsEvent(name string) istructs.IDbEvent           { panic(errNotImplemented) }
func (v *baseStateValue) RecordIDs(bool, func(string, istructs.RecordID)) { panic(errNotImplemented) }
func (v *baseStateValue) FieldNames(func(string))                         { panic(errNotImplemented) }
func (v *baseStateValue) Length() int                                     { panic(errCurrentValueIsNotAnArray) }
func (v *baseStateValue) GetAsString(int) string                          { panic(errCurrentValueIsNotAnArray) }
func (v *baseStateValue) GetAsBytes(int) []byte                           { panic(errCurrentValueIsNotAnArray) }
func (v *baseStateValue) GetAsInt32(int) int32                            { panic(errCurrentValueIsNotAnArray) }
func (v *baseStateValue) GetAsInt64(int) int64                            { panic(errCurrentValueIsNotAnArray) }
func (v *baseStateValue) GetAsFloat32(int) float32                        { panic(errCurrentValueIsNotAnArray) }
func (v *baseStateValue) GetAsFloat64(int) float64                        { panic(errCurrentValueIsNotAnArray) }
func (v *baseStateValue) GetAsQName(int) appdef.QName                     { panic(errCurrentValueIsNotAnArray) }
func (v *baseStateValue) GetAsBool(int) bool                              { panic(errCurrentValueIsNotAnArray) }
func (v *baseStateValue) GetAsValue(int) istructs.IStateValue {
	panic(errFieldByIndexIsNotAnObjectOrArray)
}

type cudsValue struct {
	istructs.IStateValue
	cuds []istructs.ICUDRow
}

func (v *cudsValue) Length() int { return len(v.cuds) }
func (v *cudsValue) GetAsValue(index int) istructs.IStateValue {
	return &cudRowValue{value: v.cuds[index]}
}

type cudRowValue struct {
	baseStateValue
	value istructs.ICUDRow
}

func (v *cudRowValue) AsInt32(name string) int32        { return v.value.AsInt32(name) }
func (v *cudRowValue) AsInt64(name string) int64        { return v.value.AsInt64(name) }
func (v *cudRowValue) AsFloat32(name string) float32    { return v.value.AsFloat32(name) }
func (v *cudRowValue) AsFloat64(name string) float64    { return v.value.AsFloat64(name) }
func (v *cudRowValue) AsBytes(name string) []byte       { return v.value.AsBytes(name) }
func (v *cudRowValue) AsString(name string) string      { return v.value.AsString(name) }
func (v *cudRowValue) AsQName(name string) appdef.QName { return v.value.AsQName(name) }
func (v *cudRowValue) AsBool(name string) bool {
	if name == sys.CUDs_Field_IsNew {
		return v.value.IsNew()
	}
	return v.value.AsBool(name)
}
func (v *cudRowValue) AsRecordID(name string) istructs.RecordID {
	return v.value.AsRecordID(name)
}
