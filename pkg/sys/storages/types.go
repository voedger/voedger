/*
  - Copyright (c) 2024-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/
package storages

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys"
)

type baseKeyBuilder struct {
	istructs.IStateKeyBuilder
	storage appdef.QName
	entity  appdef.QName
}

func (b *baseKeyBuilder) Storage() appdef.QName {
	return b.storage
}
func (b *baseKeyBuilder) Entity() appdef.QName {
	return b.entity
}
func (b *baseKeyBuilder) String() string {
	if b.entity == appdef.NullQName {
		return "storage:" + b.Storage().String()
	}
	return fmt.Sprintf("storage:%s, entity:%s", b.Storage(), b.entity.String())
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
func (b *baseKeyBuilder) PutNumber(name appdef.FieldName, value json.Number) {
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
func (b *baseValueBuilder) PutNumber(name string, value json.Number) {
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
func (v *baseStateValue) AsRecord(name string) istructs.IRecord { panic(errNotImplemented) }
func (v *baseStateValue) AsEvent(name string) istructs.IDbEvent { panic(errNotImplemented) }
func (v *baseStateValue) RecordIDs(bool) func(func(string, istructs.RecordID) bool) {
	panic(errNotImplemented)
}
func (v *baseStateValue) FieldNames(func(string) bool) { panic(errNotImplemented) }
func (v *baseStateValue) Length() int                  { panic(errCurrentValueIsNotAnArray) }
func (v *baseStateValue) GetAsString(int) string       { panic(errCurrentValueIsNotAnArray) }
func (v *baseStateValue) GetAsBytes(int) []byte        { panic(errCurrentValueIsNotAnArray) }
func (v *baseStateValue) GetAsInt32(int) int32         { panic(errCurrentValueIsNotAnArray) }
func (v *baseStateValue) GetAsInt64(int) int64         { panic(errCurrentValueIsNotAnArray) }
func (v *baseStateValue) GetAsFloat32(int) float32     { panic(errCurrentValueIsNotAnArray) }
func (v *baseStateValue) GetAsFloat64(int) float64     { panic(errCurrentValueIsNotAnArray) }
func (v *baseStateValue) GetAsQName(int) appdef.QName  { panic(errCurrentValueIsNotAnArray) }
func (v *baseStateValue) GetAsBool(int) bool           { panic(errCurrentValueIsNotAnArray) }
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

type ObjectStateValue struct {
	baseStateValue
	object istructs.IObject
}

func (v *ObjectStateValue) AsObject() istructs.IObject       { return v.object }
func (v *ObjectStateValue) AsInt32(name string) int32        { return v.object.AsInt32(name) }
func (v *ObjectStateValue) AsInt64(name string) int64        { return v.object.AsInt64(name) }
func (v *ObjectStateValue) AsFloat32(name string) float32    { return v.object.AsFloat32(name) }
func (v *ObjectStateValue) AsFloat64(name string) float64    { return v.object.AsFloat64(name) }
func (v *ObjectStateValue) AsBytes(name string) []byte       { return v.object.AsBytes(name) }
func (v *ObjectStateValue) AsString(name string) string      { return v.object.AsString(name) }
func (v *ObjectStateValue) AsQName(name string) appdef.QName { return v.object.AsQName(name) }
func (v *ObjectStateValue) AsBool(name string) bool          { return v.object.AsBool(name) }
func (v *ObjectStateValue) AsRecordID(name string) istructs.RecordID {
	return v.object.AsRecordID(name)
}
func (v *ObjectStateValue) RecordIDs(includeNulls bool) func(func(string, istructs.RecordID) bool) {
	return v.object.RecordIDs(includeNulls)
}
func (v *ObjectStateValue) FieldNames(cb func(string) bool) { v.object.FieldNames(cb) }
func (v *ObjectStateValue) AsValue(name string) (result istructs.IStateValue) {
	for n := range v.object.Containers {
		if n == name {
			result = &objectArrayContainerValue{object: v.object, container: name}
			break
		}
	}
	if result == nil {
		panic(errValueFieldUndefined(name))
	}
	return
}

type objectArrayContainerValue struct {
	baseStateValue
	object    istructs.IObject
	container string
}

func (v *objectArrayContainerValue) GetAsString(int) string      { panic(ErrNotSupported) }
func (v *objectArrayContainerValue) GetAsBytes(int) []byte       { panic(ErrNotSupported) }
func (v *objectArrayContainerValue) GetAsInt32(int) int32        { panic(ErrNotSupported) }
func (v *objectArrayContainerValue) GetAsInt64(int) int64        { panic(ErrNotSupported) }
func (v *objectArrayContainerValue) GetAsFloat32(int) float32    { panic(ErrNotSupported) }
func (v *objectArrayContainerValue) GetAsFloat64(int) float64    { panic(ErrNotSupported) }
func (v *objectArrayContainerValue) GetAsQName(int) appdef.QName { panic(ErrNotSupported) }
func (v *objectArrayContainerValue) GetAsBool(int) bool          { panic(ErrNotSupported) }
func (v *objectArrayContainerValue) GetAsValue(i int) (result istructs.IStateValue) {
	index := 0
	for o := range v.object.Children(v.container) {
		if index == i {
			result = &ObjectStateValue{object: o}
			break
		}
		index++
	}
	if result == nil {
		panic(errIndexOutOfBounds(i))
	}
	return
}
func (v *objectArrayContainerValue) Length() int {
	var result int
	for range v.object.Children(v.container) {
		result++
	}
	return result
}

type jsonArrayValue struct {
	baseStateValue
	array []interface{}
}

func (v *jsonArrayValue) GetAsString(i int) string      { return v.array[i].(string) }
func (v *jsonArrayValue) GetAsBytes(i int) []byte       { return v.array[i].([]byte) }
func (v *jsonArrayValue) GetAsInt32(i int) int32        { return v.array[i].(int32) }
func (v *jsonArrayValue) GetAsInt64(i int) int64        { return v.array[i].(int64) }
func (v *jsonArrayValue) GetAsFloat32(i int) float32    { return v.array[i].(float32) }
func (v *jsonArrayValue) GetAsFloat64(i int) float64    { return v.array[i].(float64) }
func (v *jsonArrayValue) GetAsQName(i int) appdef.QName { return v.array[i].(appdef.QName) }
func (v *jsonArrayValue) GetAsBool(i int) bool          { return v.array[i].(bool) }
func (v *jsonArrayValue) GetAsValue(i int) (result istructs.IStateValue) {
	switch v := v.array[i].(type) {
	case map[string]interface{}:
		return &jsonValue{json: v}
	case []interface{}:
		return &jsonArrayValue{array: v}
	default:
		panic(errUnexpectedType(v))
	}
}
func (v *jsonArrayValue) Length() int {
	return len(v.array)
}

type jsonValue struct {
	baseStateValue
	json map[string]interface{}
}

func (v *jsonValue) AsInt32(name string) int32 {
	if v, ok := v.json[name]; ok {
		return int32(v.(float64))
	}
	panic(errInt32FieldUndefined(name))
}
func (v *jsonValue) AsInt64(name string) int64 {
	if v, ok := v.json[name]; ok {
		return v.(int64)
	}
	panic(errInt64FieldUndefined(name))
}
func (v *jsonValue) AsFloat32(name string) float32 {
	if v, ok := v.json[name]; ok {
		return v.(float32)
	}
	panic(errFloat32FieldUndefined(name))
}
func (v *jsonValue) AsFloat64(name string) float64 {
	if v, ok := v.json[name]; ok {
		return v.(float64)
	}
	panic(errFloat64FieldUndefined(name))
}
func (v *jsonValue) AsBytes(name string) []byte {
	if v, ok := v.json[name]; ok {
		data, err := base64.StdEncoding.DecodeString(v.(string))
		if err != nil {
			panic(err)
		}
		return data
	}
	panic(errBytesFieldUndefined(name))
}
func (v *jsonValue) AsString(name string) string {
	if v, ok := v.json[name]; ok {
		return v.(string)
	}
	panic(errStringFieldUndefined(name))
}
func (v *jsonValue) AsQName(name string) appdef.QName {
	if v, ok := v.json[name]; ok {
		return appdef.MustParseQName(v.(string))
	}
	panic(errQNameFieldUndefined(name))
}
func (v *jsonValue) AsBool(name string) bool {
	if v, ok := v.json[name]; ok {
		return v.(string) == "true"
	}
	panic(errBoolFieldUndefined(name))
}
func (v *jsonValue) AsRecordID(name string) istructs.RecordID {
	if v, ok := v.json[name]; ok {
		return istructs.RecordID(v.(float64))
	}
	panic(errRecordIDFieldUndefined(name))
}
func (v *jsonValue) RecordIDs(bool) func(func(string, istructs.RecordID) bool) {
	return func(cb func(string, istructs.RecordID) bool) {}
}
func (v *jsonValue) FieldNames(cb func(string) bool) {
	for name := range v.json {
		if !cb(name) {
			break
		}
	}
}
func (v *jsonValue) AsValue(name string) (result istructs.IStateValue) {
	if v, ok := v.json[name]; ok {
		switch v := v.(type) {
		case map[string]interface{}:
			return &jsonValue{json: v}
		case []interface{}:
			return &jsonArrayValue{array: v}
		default:
			panic(errUnexpectedType(v))
		}
	}
	panic(errValueFieldUndefined(name))
}

type wsTypeKey struct {
	wsid     istructs.WSID
	appQName appdef.AppQName
}

// implements iStructureInt64FieldTypeChecker, iViewInt64FieldTypeChecker
type wsTypeVailidator struct {
	appStructsFunc state.AppStructsFunc
	wsidKinds      map[wsTypeKey]appdef.QName
}

func newWsTypeValidator(appStructsFunc state.AppStructsFunc) wsTypeVailidator {
	return wsTypeVailidator{
		appStructsFunc: appStructsFunc,
		wsidKinds:      make(map[wsTypeKey]appdef.QName),
	}
}

func (v *wsTypeVailidator) isStructureInt64FieldRecordID(name appdef.QName, fieldName appdef.FieldName) bool {
	app := v.appStructsFunc().AppDef()
	rec := app.Structure(name)
	field := rec.Field(fieldName)
	if field == nil {
		panic(errInt64FieldUndefined(fieldName))
	}
	return field.DataKind() == appdef.DataKind_RecordID
}

func (v *wsTypeVailidator) isViewInt64FieldRecordID(name appdef.QName, fieldName appdef.FieldName) bool {
	app := v.appStructsFunc().AppDef()
	rec := app.View(name)
	field := rec.Field(fieldName)
	if field == nil {
		panic(errInt64FieldUndefined(fieldName))
	}
	return field.DataKind() == appdef.DataKind_RecordID
}

// Returns NullQName if not found
func (v *wsTypeVailidator) getWSIDKind(wsid istructs.WSID, entity appdef.QName) (appdef.QName, error) {
	key := wsTypeKey{wsid: wsid, appQName: v.appStructsFunc().AppQName()}
	wsKind, ok := v.wsidKinds[key]
	if !ok {
		wsDesc, err := v.appStructsFunc().Records().GetSingleton(wsid, qNameCDocWorkspaceDescriptor)
		if err != nil {
			// notest
			return appdef.NullQName, err
		}
		if wsDesc.QName() == appdef.NullQName {
			if v.appStructsFunc().AppDef().WorkspaceByDescriptor(entity) != nil {
				// Special case. sys.CreateWorkspace creates WSKind while WorkspaceDescriptor is not applied yet.
				return entity, nil
			}
			return appdef.NullQName, fmt.Errorf("%w: %d", errWorkspaceDescriptorNotFound, wsid)
		}
		wsKind = wsDesc.AsQName(field_WSKind)
		if len(v.wsidKinds) < wsidTypeValidatorCacheSize {
			v.wsidKinds[key] = wsKind
		}
	}
	return wsKind, nil
}

func (v *wsTypeVailidator) validate(wsid istructs.WSID, entity appdef.QName) error {
	if entity == qNameCDocWorkspaceDescriptor {
		return nil // This QName always can be read and write. Otherwise sys.CreateWorkspace is not able to create descriptor.
	}
	if wsid != istructs.NullWSID && v.appStructsFunc().Records() != nil { // NullWSID only stores actualizer offsets
		wsKind, err := v.getWSIDKind(wsid, entity)
		if err != nil {
			// notest
			return err
		}
		ws := v.appStructsFunc().AppDef().WorkspaceByDescriptor(wsKind)
		if ws == nil {
			// notest
			return errDescriptorForUndefinedWorkspace
		}
		if ws.TypeByName(entity) == nil {
			return typeIsNotDefinedInWorkspaceWithDescriptor(entity, wsKind)
		}
	}
	return nil
}
