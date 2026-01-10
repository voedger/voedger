/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */

package storages

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"reflect"
	"sort"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys"
)

const fmtErrHashCode = "mkb.HashCode(): %w"

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
	valueBuilders   map[uint64]istructs.IStateValueBuilder
}

func (s *MockedStorage) NewKeyBuilder(
	entity appdef.QName,
	existingBuilder istructs.IStateKeyBuilder,
) istructs.IStateKeyBuilder {
	kb := newMockedKeyBuilder(s, entity)
	kb.IsNew_ = true

	if existingBuilder != nil {
		kb.ID_ = existingBuilder.(*mockedKeyBuilder).ID_
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

		hashCode, err := mkb.HashCode()
		if err != nil {
			return fmt.Errorf(fmtErrHashCode, err)
		}

		vb, ok := s.valueBuilders[hashCode]
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

	hashCode, err := mkb.HashCode()
	if err != nil {
		return nil, fmt.Errorf(fmtErrHashCode, err)
	}

	vb, ok := s.valueBuilders[hashCode]
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

	hashCode, err := mkb.HashCode()
	if err != nil {
		return fmt.Errorf(fmtErrHashCode, err)
	}

	vb, ok := s.valueBuilders[hashCode]
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

	hashCode, err := mkb.HashCode()
	if err != nil {
		return nil, fmt.Errorf(fmtErrHashCode, err)
	}

	s.valueBuilders[hashCode] = newMVB

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

func (s *MockedStorage) PutViewRecord(kb istructs.IStateKeyBuilder, value *coreutils.TestObject) error {
	mkb, ok := kb.(*mockedKeyBuilder)
	if !ok {
		panic(errMockedKeyBuilderExpected)
	}

	hashCode, err := mkb.HashCode()
	if err != nil {
		return fmt.Errorf(fmtErrHashCode, err)
	}

	s.keyBuilders = append(s.keyBuilders, mkb)
	s.valueBuilders[hashCode] = newMockedValueBuilder(newMockedStateValue([]*coreutils.TestObject{value}))

	return nil
}

func (s *MockedStorage) PutRecord(recordID uint64, value *coreutils.TestObject) {
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
		valueBuilders:   make(map[uint64]istructs.IStateValueBuilder),
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

// HashCode returns a hash code for the key builder.
// It is used to identify a value associated to key builder in the storage.
func (mkb *mockedKeyBuilder) HashCode() (uint64, error) {
	if mkb.ID_ == 0 && len(mkb.Data) > 0 {
		// Convert map to a sorted slice of key-value pairs to ensure consistent ordering
		var pairs []struct {
			Key   string
			Value any
		}
		for k, v := range mkb.Data {
			pairs = append(pairs, struct {
				Key   string
				Value any
			}{Key: k, Value: v})
		}

		sort.Slice(pairs, func(i, j int) bool {
			return pairs[i].Key < pairs[j].Key
		})

		// Serialize to JSON
		data, err := json.Marshal(pairs)
		if err != nil {
			return 0, fmt.Errorf("json.Marshal(): %w", err)
		}

		// Compute the FNV-1a hash
		hasher := fnv.New64a()
		if _, err = hasher.Write(data); err != nil {
			return 0, fmt.Errorf("hasher.Write(): %w", err)
		}

		return hasher.Sum64(), nil
	}

	return uint64(mkb.ID_), nil
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

	if len(mkb.Data) != len(m.Data) {
		return false
	}

	for k, v1 := range mkb.Data {
		v2, exists := m.Data[k]
		if !exists {
			return false
		}

		if jsonNumber, ok := v2.(json.Number); ok {
			v2, err := jsonNumber.Int64()
			if err != nil {
				panic(fmt.Errorf("jsonNumber.Int64(): %w", err))
			}

			switch t1 := v1.(type) {
			case int8:
				//nolint:gosec
				if t1 != int8(v2) {
					return false
				}
			case int16:
				//nolint:gosec
				if t1 != int16(v2) {
					return false
				}
			case int32:
				//nolint:gosec
				if t1 != int32(v2) {
					return false
				}
			case int64:
				if t1 != v2 {
					return false
				}
			default:
				if !reflect.DeepEqual(t1, v2) {
					return false
				}
			}

			continue
		}

		if !reflect.DeepEqual(v1, v2) {
			return false
		}
	}

	return mkb.Name == m.Name && mkb.ID_ == m.ID_
}

func (mkb *mockedKeyBuilder) String() string {
	return mkb.baseKeyBuilder.String() + fmt.Sprintf(
		", mockedKeyBuilder - qname: %s, id: %d",
		mkb.TestObject.Name.String(),
		mkb.TestObject.ID_,
	)
}

func (mkb *mockedKeyBuilder) ToBytes(istructs.WSID) ([]byte, []byte, error) { return nil, nil, nil }

func (mkb *mockedKeyBuilder) PutInt8(field appdef.FieldName, value int8) {
	mkb.TestObject.Data[field] = value
}

func (mkb *mockedKeyBuilder) PutInt16(field appdef.FieldName, value int16) {
	mkb.TestObject.Data[field] = value
}

func (mkb *mockedKeyBuilder) PutInt32(field appdef.FieldName, value int32) {
	mkb.TestObject.Data[field] = value
}

func (mkb *mockedKeyBuilder) PutInt64(field appdef.FieldName, value int64) {
	mkb.TestObject.Data[field] = value
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
	// if IsSingleton is true, then ID must be set to FirstSingletonID
	// it is workaround for singleton entities
	if field == sys.Storage_Record_Field_IsSingleton && value {
		mkb.TestObject.ID_ = mkb.mockedStorage.GetSingletonID(mkb.Name)
	}
}

func (mkb *mockedKeyBuilder) PutRecordID(field appdef.FieldName, value istructs.RecordID) {
	if field == sys.Storage_Record_Field_ID {
		//nolint:gosec
		mkb.TestObject.ID_ = value

		return
	}

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

func (mvb *mockedValueBuilder) PutInt8(name appdef.FieldName, i int8) {
	mvb.value.TestObjects[0].Data[name] = i
}

func (mvb *mockedValueBuilder) PutInt16(name appdef.FieldName, i int16) {
	mvb.value.TestObjects[0].Data[name] = i
}

func (mvb *mockedValueBuilder) PutInt32(name appdef.FieldName, i int32) {
	mvb.value.TestObjects[0].Data[name] = i
}

func (mvb *mockedValueBuilder) PutInt64(name appdef.FieldName, i int64) {
	if name == appdef.SystemField_ID {
		//nolint:gosec
		mvb.value.TestObjects[0].ID_ = istructs.RecordID(i)

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
	if name == appdef.SystemField_ID {
		mvb.value.TestObjects[0].ID_ = id
	} else {
		mvb.value.TestObjects[0].Data[name] = id
	}
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
			newValue.TestObjects[i].ID_ = value[i].ID_
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
					copy(newValue.TestObjects[i].Containers_[k], v)
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

func (m *mockedStateValue) AsInt8(name appdef.FieldName) int8 {
	switch t := m.TestObjects[0].Data[name].(type) {
	case int8:
		return t
	case json.Number:
		t2, err := coreutils.ClarifyJSONNumber(t, appdef.DataKind_int8)
		if err != nil {
			panic(err)
		}

		return t2.(int8)
	default:
		panic(fmt.Sprintf("mockedStateValue.AsInt8(%s): unexpected type", name))
	}
}

func (m *mockedStateValue) AsInt16(name appdef.FieldName) int16 {
	switch t := m.TestObjects[0].Data[name].(type) {
	case int16:
		return t
	case json.Number:
		t2, err := coreutils.ClarifyJSONNumber(t, appdef.DataKind_int16)
		if err != nil {
			panic(err)
		}

		return t2.(int16)
	default:
		panic(fmt.Sprintf("mockedStateValue.AsInt16(%s): unexpected type", name))
	}
}

func (m *mockedStateValue) AsInt32(name appdef.FieldName) int32 {
	switch t := m.TestObjects[0].Data[name].(type) {
	case int32:
		return t
	case json.Number:
		t2, err := coreutils.ClarifyJSONNumber(t, appdef.DataKind_int32)
		if err != nil {
			panic(err)
		}

		return t2.(int32)
	default:
		panic(fmt.Sprintf("mockedStateValue.AsInt32(%s): unexpected type", name))
	}
}

func (m *mockedStateValue) AsInt64(name appdef.FieldName) int64 {
	switch t := m.TestObjects[0].Data[name].(type) {
	case int:
		return int64(t)
	case int64:
		return t
	case istructs.Offset:
		//nolint:gosec
		return int64(t)
	case istructs.RecordID:
		//nolint:gosec
		return int64(t)
	case json.Number:
		t2, err := coreutils.ClarifyJSONNumber(t, appdef.DataKind_int64)
		if err != nil {
			panic(err)
		}

		return t2.(int64)
	default:
		panic(fmt.Sprintf("mockedStateValue.AsInt64(%s): unexpected type", name))
	}
}

func (m *mockedStateValue) AsFloat32(name appdef.FieldName) float32 {
	switch t := m.TestObjects[0].Data[name].(type) {
	case float32:
		return t
	case json.Number:
		t2, err := coreutils.ClarifyJSONNumber(t, appdef.DataKind_float32)
		if err != nil {
			panic(err)
		}

		return t2.(float32)
	default:
		panic(fmt.Sprintf("mockedStateValue.AsFloat32(%s): unexpected type", name))
	}
}

func (m *mockedStateValue) AsFloat64(name appdef.FieldName) float64 {
	switch t := m.TestObjects[0].Data[name].(type) {
	case float64:
		return t
	case json.Number:
		t2, err := coreutils.ClarifyJSONNumber(t, appdef.DataKind_float64)
		if err != nil {
			panic(err)
		}

		return t2.(float64)
	default:
		panic(fmt.Sprintf("mockedStateValue.AsFloat64(%s): unexpected type", name))
	}
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
		return m.TestObjects[0].ID_
	}

	switch t := m.TestObjects[0].Data[name].(type) {
	case int64:
		//nolint:gosec
		return istructs.RecordID(t)
	case istructs.RecordID:
		return t
	case json.Number:
		res, err := coreutils.ClarifyJSONNumber(t, appdef.DataKind_RecordID)
		if err != nil {
			panic(err)
		}
		return res.(istructs.RecordID)
	default:
		panic(fmt.Sprintf("mockedStateValue.AsRecordID(%s): unexpected type", name))
	}
}

func (m *mockedStateValue) RecordIDs(includeNulls bool) func(func(appdef.FieldName, istructs.RecordID) bool) {
	panic(errNotImplemented)
}

func (m *mockedStateValue) SpecifiedValues(f func(appdef.IField, any) bool) {
	panic(errNotImplemented)
}

func (m *mockedStateValue) Fields(f func(appdef.IField) bool) {
	panic(errNotImplemented)
}

func (m *mockedStateValue) AsValue(name string) istructs.IStateValue {
	v, ok := m.TestObjects[0].Containers_[name]
	if ok {
		msv := &mockedStateValue{
			TestObjects: make([]*coreutils.TestObject, len(v)),
		}

		copy(msv.TestObjects, v)

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
