/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */

package storages

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys"
)

type MockedStorage struct {
	state.IWithInsert
	state.IWithGet
	state.IWithUpdate
	state.IWithGetBatch
	state.IWithRead
	state.IWithApplyBatch

	nextSingletonID istructs.RecordID
	singletonIDs    map[appdef.QName]istructs.RecordID
	storageQName    appdef.QName
	keyBuilders     []*mockedKeyBuilder
	valueBuilders   map[istructs.RecordID]istructs.IStateValueBuilder
}

func (s *MockedStorage) NewKeyBuilder(
	entity appdef.QName,
	existingBuilder istructs.IStateKeyBuilder,
) istructs.IStateKeyBuilder {
	kb := newMockedKeyBuilder(s, entity)
	kb.IsNew_ = true

	if existingBuilder != nil {
		kb.Id = existingBuilder.(*mockedKeyBuilder).Id
		kb.IsNew_ = false
	}

	return kb
}

func (s *MockedStorage) GetBatch(items []state.GetBatchItem) (err error) {
	for _, item := range items {
		mkb, ok := item.Key.(*mockedKeyBuilder)
		if !ok {
			return errMockedKeyBuilderExpected
		}

		vb, ok := s.valueBuilders[mkb.Id]
		if !ok {
			return ErrNotFoundKey
		}

		item.Value = vb.BuildValue()
	}

	return nil
}

func (s *MockedStorage) Get(kb istructs.IStateKeyBuilder) (value istructs.IStateValue, err error) {
	mkb, ok := kb.(*mockedKeyBuilder)
	if !ok {
		return nil, errMockedKeyBuilderExpected
	}

	vb, ok := s.valueBuilders[mkb.Id]
	if !ok {
		return nil, nil
	}

	return vb.BuildValue(), nil
}

func (s *MockedStorage) Read(kb istructs.IStateKeyBuilder, callback istructs.ValueCallback) (err error) {
	mkb, ok := kb.(*mockedKeyBuilder)
	if !ok {
		return errMockedKeyBuilderExpected
	}

	vb, ok := s.valueBuilders[mkb.Id]
	if !ok {
		return ErrNotFoundKey
	}

	return callback(mkb.Key(), vb.BuildValue())
}

func (s *MockedStorage) ProvideValueBuilder(
	kb istructs.IStateKeyBuilder,
	existingValueBuilder istructs.IStateValueBuilder,
) (istructs.IStateValueBuilder, error) {
	var value istructs.IStateValue
	if existingValueBuilder != nil {
		value = existingValueBuilder.BuildValue()
	}

	mkb, ok := kb.(*mockedKeyBuilder)
	if !ok {
		return nil, errMockedKeyBuilderExpected
	}

	newMVB := newMockedValueBuilder(value)
	s.keyBuilders = append(s.keyBuilders, mkb)
	s.valueBuilders[mkb.Id] = newMVB

	return newMVB, nil
}

func (s *MockedStorage) ProvideValueBuilderForUpdate(
	_ istructs.IStateKeyBuilder,
	existingValue istructs.IStateValue,
	existingValueBuilder istructs.IStateValueBuilder,
) (istructs.IStateValueBuilder, error) {
	if existingValueBuilder != nil {
		return newMockedValueBuilder(existingValueBuilder.BuildValue()), nil
	}

	return newMockedValueBuilder(existingValue), nil
}

func (s *MockedStorage) Validate([]state.ApplyBatchItem) (err error) {
	return
}

func (s *MockedStorage) ApplyBatch([]state.ApplyBatchItem) (err error) {
	return
}

func (s *MockedStorage) PutValue(recordID istructs.RecordID, value *coreutils.TestObject) {
	s.valueBuilders[recordID] = newMockedValueBuilder(newMockedStateValue([]*coreutils.TestObject{value}))
}

func (s *MockedStorage) GetSingletonID(singletonQName appdef.QName) istructs.RecordID {
	id, ok := s.singletonIDs[singletonQName]
	if !ok {
		id = s.nextSingletonID
		s.singletonIDs[singletonQName] = id
		s.nextSingletonID++
	}

	return id
}

func NewMockedStorage(storageQName appdef.QName) *MockedStorage {
	return &MockedStorage{
		storageQName:    storageQName,
		keyBuilders:     make([]*mockedKeyBuilder, 0),
		valueBuilders:   make(map[istructs.RecordID]istructs.IStateValueBuilder),
		singletonIDs:    make(map[appdef.QName]istructs.RecordID),
		nextSingletonID: istructs.FirstSingletonID,
	}
}

type mockedKeyBuilder struct {
	baseKeyBuilder
	mockedStorage *MockedStorage
	coreutils.TestObject
}

func newMockedKeyBuilder(mockedStorage *MockedStorage, entity appdef.QName) *mockedKeyBuilder {
	return &mockedKeyBuilder{
		TestObject: coreutils.TestObject{
			Name: entity,
			Data: make(map[string]any),
		},
		baseKeyBuilder: baseKeyBuilder{
			storage: mockedStorage.storageQName,
			entity:  entity,
		},
		mockedStorage: mockedStorage,
	}
}

func (mkb *mockedKeyBuilder) Key() istructs.IKey {
	return &mkb.TestObject
}

func (mkb *mockedKeyBuilder) PartitionKey() istructs.IRowWriter {
	return &mkb.TestObject
}

func (mkb *mockedKeyBuilder) ClusteringColumns() istructs.IRowWriter {
	return &mkb.TestObject
}

func (mkb *mockedKeyBuilder) Storage() appdef.QName {
	return mkb.storage
}

func (mkb *mockedKeyBuilder) Entity() appdef.QName {
	return mkb.entity
}

func (mkb *mockedKeyBuilder) Equals(kb istructs.IKeyBuilder) bool {
	m, ok := kb.(*mockedKeyBuilder)
	if !ok {
		return false
	}

	return mkb.Name == m.Name && mkb.Id == m.Id
}

func (mkb *mockedKeyBuilder) String() string {
	return mkb.baseKeyBuilder.String() + fmt.Sprintf(
		", mockedKeyBuilder - qname: %s, id: %d",
		mkb.TestObject.Name.String(),
		mkb.TestObject.Id,
	)
}

func (mkb *mockedKeyBuilder) ToBytes(istructs.WSID) ([]byte, []byte, error) { return nil, nil, nil }

func (mkb *mockedKeyBuilder) PutInt32(field appdef.FieldName, value int32) {
	mkb.TestObject.Data[field] = value
}

func (mkb *mockedKeyBuilder) PutInt64(field appdef.FieldName, value int64) {
	mkb.TestObject.Data[field] = value
	if field == sys.Storage_Record_Field_ID {
		//nolint:gosec
		mkb.TestObject.Id = istructs.RecordID(value)
	}
}

func (mkb *mockedKeyBuilder) PutFloat32(field appdef.FieldName, value float32) {
	mkb.TestObject.Data[field] = value
}

func (mkb *mockedKeyBuilder) PutFloat64(field appdef.FieldName, value float64) {
	mkb.TestObject.Data[field] = value
}

func (mkb *mockedKeyBuilder) PutBytes(field appdef.FieldName, value []byte) {
	mkb.TestObject.Data[field] = value
}

func (mkb *mockedKeyBuilder) PutString(field appdef.FieldName, value string) {
	mkb.TestObject.Data[field] = value
}

func (mkb *mockedKeyBuilder) PutQName(field appdef.FieldName, value appdef.QName) {
	mkb.TestObject.Data[field] = value
}

func (mkb *mockedKeyBuilder) PutBool(field appdef.FieldName, value bool) {
	mkb.TestObject.Data[field] = value
	// if IsSingleton is true, then ID must be set to MinReservedBaseRecordID
	// it is workaround for singleton entities
	if field == sys.Storage_Record_Field_IsSingleton && value {
		mkb.TestObject.Id = mkb.mockedStorage.GetSingletonID(mkb.Name)
	}
}

func (mkb *mockedKeyBuilder) PutRecordID(field appdef.FieldName, value istructs.RecordID) {
	mkb.TestObject.Data[field] = value
}

func (mkb *mockedKeyBuilder) PutNumber(field appdef.FieldName, value json.Number) {
	mkb.TestObject.Data[field] = value
}

func (mkb *mockedKeyBuilder) PutChars(field appdef.FieldName, value string) {
	mkb.TestObject.Data[field] = value
}

func (mkb *mockedKeyBuilder) PutFromJSON(value map[appdef.FieldName]any) {
	for k, v := range value {
		mkb.TestObject.Data[k] = v
	}
}

type mockedValueBuilder struct {
	value *mockedStateValue
}

func (mvb *mockedValueBuilder) PutInt32(name appdef.FieldName, i int32) {
	mvb.value.TestObjects[0].Data[name] = i
}

func (mvb *mockedValueBuilder) PutInt64(name appdef.FieldName, i int64) {
	if name == appdef.SystemField_ID {
		mvb.value.TestObjects[0].Id = istructs.RecordID(i)

		return
	}

	mvb.value.TestObjects[0].Data[name] = i
}

func (mvb *mockedValueBuilder) PutFloat32(name appdef.FieldName, f float32) {
	mvb.value.TestObjects[0].Data[name] = f
}

func (mvb *mockedValueBuilder) PutFloat64(name appdef.FieldName, f float64) {
	mvb.value.TestObjects[0].Data[name] = f
}

func (mvb *mockedValueBuilder) PutBytes(name appdef.FieldName, bytes []byte) {
	mvb.value.TestObjects[0].Data[name] = bytes
}

func (mvb *mockedValueBuilder) PutString(name appdef.FieldName, s string) {
	mvb.value.TestObjects[0].Data[name] = s
}

func (mvb *mockedValueBuilder) PutQName(name appdef.FieldName, qName appdef.QName) {
	mvb.value.TestObjects[0].Data[name] = qName
}

func (mvb *mockedValueBuilder) PutBool(name appdef.FieldName, b bool) {
	mvb.value.TestObjects[0].Data[name] = b
}

func (mvb *mockedValueBuilder) PutRecordID(name appdef.FieldName, id istructs.RecordID) {
	mvb.value.TestObjects[0].Data[name] = id
}

func (mvb *mockedValueBuilder) PutNumber(name appdef.FieldName, number json.Number) {
	mvb.value.TestObjects[0].Data[name] = number
}

func (mvb *mockedValueBuilder) PutChars(name appdef.FieldName, s string) {
	mvb.value.TestObjects[0].Data[name] = s
}

func (mvb *mockedValueBuilder) PutFromJSON(m map[appdef.FieldName]any) {
	for name, value := range m {
		mvb.value.TestObjects[0].Data[name] = value
	}
}

func newMockedValueBuilder(value istructs.IStateValue) (mvb *mockedValueBuilder) {
	if value == nil {
		return &mockedValueBuilder{
			value: newMockedStateValue(
				[]*coreutils.TestObject{
					{
						Data:        make(map[string]any),
						Containers_: make(map[string][]*coreutils.TestObject),
					},
				},
			),
		}
	}

	mv, ok := value.(*mockedStateValue)
	if !ok {
		panic("newMockedValueBuilder: value is not a mockedValue")
	}

	return &mockedValueBuilder{
		value: mv,
	}
}

func (mvb *mockedValueBuilder) Equal(to istructs.IStateValueBuilder) bool {
	vb, ok := to.(*mockedValueBuilder)
	if !ok {
		return false
	}

	return reflect.DeepEqual(mvb.value, vb.value)
}

func newMockedStateValue(value []*coreutils.TestObject) *mockedStateValue {
	newValue := &mockedStateValue{
		TestObjects: make([]*coreutils.TestObject, len(value)),
	}

	for i := 0; i < len(value); i++ {
		newValue.TestObjects[i] = &coreutils.TestObject{
			Data:        make(map[string]any),
			Containers_: make(map[string][]*coreutils.TestObject),
		}

		if value[i] != nil {
			newValue.TestObjects[i].Name = value[i].Name
			newValue.TestObjects[i].Id = value[i].Id
			newValue.TestObjects[i].Parent_ = value[i].Parent_
			newValue.TestObjects[i].IsNew_ = value[i].IsNew_

			if value[i].Data != nil {
				for k, v := range value[i].Data {
					newValue.TestObjects[i].Data[k] = v
				}
			}

			if value[i].Containers_ != nil {
				for k, v := range value[i].Containers_ {
					newValue.TestObjects[i].Containers_[k] = make([]*coreutils.TestObject, len(v))
					for kk, vv := range v {
						newValue.TestObjects[i].Containers_[k][kk] = vv
					}
				}
			}
		}

	}

	return newValue
}

func (mvb *mockedValueBuilder) BuildValue() istructs.IStateValue {
	return mvb.value
}

type mockedStateValue struct {
	TestObjects []*coreutils.TestObject
}

func (m *mockedStateValue) AsInt32(name appdef.FieldName) int32 {
	switch t := m.TestObjects[0].Data[name].(type) {
	case int32:
		return t
	case json.Number:
		val, err := t.Int64()
		if err != nil {
			panic(fmt.Errorf("json.Number.AsInt64(): %w", err))
		}
		//nolint:gosec
		return int32(val)
	default:
		panic(fmt.Sprintf("mockedStateValue.AsInt32(%s): unexpected type", name))
	}
}

func (m *mockedStateValue) AsInt64(name appdef.FieldName) int64 {
	switch t := m.TestObjects[0].Data[name].(type) {
	case int64:
		return t
	case json.Number:
		val, err := t.Int64()
		if err != nil {
			panic(fmt.Errorf("json.Number.AsInt64(): %w", err))
		}

		return val
	default:
		panic(fmt.Sprintf("mockedStateValue.AsInt64(%s): unexpected type", name))
	}
}

func (m *mockedStateValue) AsFloat32(name appdef.FieldName) float32 {
	return m.TestObjects[0].Data[name].(float32)
}

func (m *mockedStateValue) AsFloat64(name appdef.FieldName) float64 {
	return m.TestObjects[0].Data[name].(float64)
}

func (m *mockedStateValue) AsBytes(name appdef.FieldName) []byte {
	return m.TestObjects[0].Data[name].([]byte)
}

func (m *mockedStateValue) AsString(name appdef.FieldName) string {
	return m.TestObjects[0].Data[name].(string)
}

func (m *mockedStateValue) AsQName(name appdef.FieldName) appdef.QName {
	return m.TestObjects[0].Name
}

func (m *mockedStateValue) AsBool(name appdef.FieldName) bool {
	return m.TestObjects[0].Data[name].(bool)
}

func (m *mockedStateValue) AsRecordID(name appdef.FieldName) istructs.RecordID {
	if name == appdef.SystemField_ID {
		return m.TestObjects[0].Id
	}

	switch t := m.TestObjects[0].Data[name].(type) {
	case int32:
		return istructs.RecordID(t)
	case int64:
		return istructs.RecordID(t)
	case istructs.RecordID:
		return t
	default:
		panic(fmt.Sprintf("mockedStateValue.AsRecordID(%s): unexpected type", name))
	}
}

func (m *mockedStateValue) RecordIDs(includeNulls bool) func(func(appdef.FieldName, istructs.RecordID) bool) {
	panic(errNotImplemented)
}

func (m *mockedStateValue) FieldNames(f func(appdef.FieldName) bool) {
	panic(errNotImplemented)
}

func (m *mockedStateValue) AsValue(name string) istructs.IStateValue {
	v, ok := m.TestObjects[0].Containers_[name]
	if ok {
		msv := &mockedStateValue{
			TestObjects: make([]*coreutils.TestObject, len(v)),
		}

		for i := 0; i < len(v); i++ {
			msv.TestObjects[i] = v[i]
		}

		return msv
	}

	return nil
}

func (m *mockedStateValue) Length() int {
	return len(m.TestObjects)
}

func (m mockedStateValue) GetAsString(index int) string {
	if index < 0 || index >= len(m.TestObjects) {
		panic(fmt.Sprintf("mockedStateValue.GetAsString(%d): index out of range", index))
	}

	panic(errNotImplemented)
}

func (m mockedStateValue) GetAsBytes(index int) []byte {
	panic(errNotImplemented)
}

func (m mockedStateValue) GetAsInt32(index int) int32 {
	panic(errNotImplemented)
}

func (m mockedStateValue) GetAsInt64(index int) int64 {
	panic(errNotImplemented)
}

func (m mockedStateValue) GetAsFloat32(index int) float32 {
	panic(errNotImplemented)
}

func (m mockedStateValue) GetAsFloat64(index int) float64 {
	panic(errNotImplemented)
}

func (m mockedStateValue) GetAsQName(index int) appdef.QName {
	panic(errNotImplemented)
}

func (m mockedStateValue) GetAsBool(index int) bool {
	panic(errNotImplemented)
}

func (m mockedStateValue) GetAsValue(index int) istructs.IStateValue {
	if index < 0 || index >= len(m.TestObjects) {
		panic(fmt.Sprintf("mockedStateValue.GetAsValue(%d): index out of range", index))
	}

	return newMockedStateValue([]*coreutils.TestObject{m.TestObjects[index]})
}
