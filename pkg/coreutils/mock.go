/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 *
 * @author Daniil Solovyov
 */

// This is AI generated code do not edit it manually

package coreutils

import (
	"encoding/json"

	"github.com/stretchr/testify/mock"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

type MockCUDRow struct {
	mock.Mock
}

func (m *MockCUDRow) AsInt32(name appdef.FieldName) int32     { return m.Called(name).Get(0).(int32) }
func (m *MockCUDRow) AsInt64(name appdef.FieldName) int64     { return m.Called(name).Get(0).(int64) }
func (m *MockCUDRow) AsFloat32(name appdef.FieldName) float32 { return m.Called(name).Get(0).(float32) }
func (m *MockCUDRow) AsFloat64(name appdef.FieldName) float64 { return m.Called(name).Get(0).(float64) }
func (m *MockCUDRow) AsBytes(name appdef.FieldName) []byte    { return m.Called(name).Get(0).([]byte) }
func (m *MockCUDRow) AsString(name appdef.FieldName) string   { return m.Called(name).Get(0).(string) }
func (m *MockCUDRow) AsQName(name appdef.FieldName) appdef.QName {
	return m.Called(name).Get(0).(appdef.QName)
}
func (m *MockCUDRow) AsBool(name appdef.FieldName) bool         { return m.Called(name).Get(0).(bool) }
func (m *MockCUDRow) FieldNames(cb func(appdef.FieldName) bool) { m.Called(cb) }
func (m *MockCUDRow) IsActivated() bool                         { return m.Called().Get(0).(bool) }
func (m *MockCUDRow) IsDeactivated() bool                       { return m.Called().Get(0).(bool) }
func (m *MockCUDRow) IsNew() bool                               { return m.Called().Get(0).(bool) }
func (m *MockCUDRow) QName() appdef.QName                       { return m.Called().Get(0).(appdef.QName) }
func (m *MockCUDRow) ID() istructs.RecordID                     { return m.Called().Get(0).(istructs.RecordID) }
func (m *MockCUDRow) ModifiedFields(cb func(appdef.FieldName, interface{}) bool) {
	m.Called(cb)
}
func (m *MockCUDRow) AsRecordID(name appdef.FieldName) istructs.RecordID {
	return m.Called(name).Get(0).(istructs.RecordID)
}
func (m *MockCUDRow) RecordIDs(includeNulls bool) func(func(appdef.FieldName, istructs.RecordID) bool) {
	return m.Called(includeNulls).Get(0).(func(func(appdef.FieldName, istructs.RecordID) bool))
}

type MockPLogEvent struct {
	mock.Mock
}

func (m *MockPLogEvent) ArgumentObject() istructs.IObject {
	return m.Called().Get(0).(istructs.IObject)
}
func (m *MockPLogEvent) Bytes() []byte { return m.Called().Get(0).([]byte) }
func (m *MockPLogEvent) CUDs(cb func(rec istructs.ICUDRow) bool) {
	m.Called(cb)
}
func (m *MockPLogEvent) RegisteredAt() istructs.UnixMilli {
	return m.Called().Get(0).(istructs.UnixMilli)
}
func (m *MockPLogEvent) DeviceID() istructs.ConnectedDeviceID {
	return m.Called().Get(0).(istructs.ConnectedDeviceID)
}
func (m *MockPLogEvent) Synced() bool                 { return m.Called().Bool(0) }
func (m *MockPLogEvent) QName() appdef.QName          { return m.Called().Get(0).(appdef.QName) }
func (m *MockPLogEvent) SyncedAt() istructs.UnixMilli { return m.Called().Get(0).(istructs.UnixMilli) }
func (m *MockPLogEvent) Error() istructs.IEventError  { return m.Called().Get(0).(istructs.IEventError) }
func (m *MockPLogEvent) Workspace() istructs.WSID     { return m.Called().Get(0).(istructs.WSID) }
func (m *MockPLogEvent) WLogOffset() istructs.Offset  { return m.Called().Get(0).(istructs.Offset) }
func (m *MockPLogEvent) Release()                     { m.Called() }

type MockObject struct {
	mock.Mock
}

func (m *MockObject) AsInt32(name appdef.FieldName) int32     { return m.Called(name).Get(0).(int32) }
func (m *MockObject) AsInt64(name appdef.FieldName) int64     { return m.Called(name).Get(0).(int64) }
func (m *MockObject) AsFloat32(name appdef.FieldName) float32 { return m.Called(name).Get(0).(float32) }
func (m *MockObject) AsFloat64(name appdef.FieldName) float64 { return m.Called(name).Get(0).(float64) }
func (m *MockObject) AsBytes(name appdef.FieldName) []byte    { return m.Called(name).Get(0).([]byte) }
func (m *MockObject) AsString(name appdef.FieldName) string   { return m.Called(name).Get(0).(string) }
func (m *MockObject) AsQName(name appdef.FieldName) appdef.QName {
	return m.Called(name).Get(0).(appdef.QName)
}
func (m *MockObject) AsBool(name appdef.FieldName) bool         { return m.Called(name).Get(0).(bool) }
func (m *MockObject) QName() appdef.QName                       { return m.Called().Get(0).(appdef.QName) }
func (m *MockObject) AsRecord() istructs.IRecord                { return m.Called().Get(0).(istructs.IRecord) }
func (m *MockObject) Containers(cb func(string) bool)           { m.Called(cb) }
func (m *MockObject) FieldNames(cb func(appdef.FieldName) bool) { m.Called(cb) }
func (m *MockObject) AsRecordID(name appdef.FieldName) istructs.RecordID {
	return m.Called(name).Get(0).(istructs.RecordID)
}
func (m *MockObject) RecordIDs(includeNulls bool) func(func(appdef.FieldName, istructs.RecordID) bool) {
	return m.Called(includeNulls).Get(0).(func(func(appdef.FieldName, istructs.RecordID) bool))
}
func (m *MockObject) Children(container ...string) func(func(istructs.IObject) bool) {
	args := m.Called(container)
	return args.Get(0).(func(func(istructs.IObject) bool))
}

type MockState struct {
	mock.Mock
}

func (m *MockState) App() appdef.AppQName {
	args := m.Called()
	return args.Get(0).(appdef.AppQName)
}

func (m *MockState) AppStructs() istructs.IAppStructs {
	args := m.Called()
	return args.Get(0).(istructs.IAppStructs)
}

func (m *MockState) CanExist(key istructs.IStateKeyBuilder) (value istructs.IStateValue, ok bool, err error) {
	args := m.Called(key)
	if intf := args.Get(0); intf != nil {
		value = intf.(istructs.IStateValue)
	}
	return value, args.Bool(1), args.Error(2)
}
func (m *MockState) CanExistAll(keys []istructs.IStateKeyBuilder, callback istructs.StateValueCallback) (err error) {
	args := m.Called(keys, callback)
	return args.Error(0)
}
func (m *MockState) CommandPrepareArgs() istructs.CommandPrepareArgs {
	args := m.Called()
	return args.Get(0).(istructs.CommandPrepareArgs)
}
func (m *MockState) KeyBuilder(storage, entity appdef.QName) (builder istructs.IStateKeyBuilder, err error) {
	args := m.Called(storage, entity)
	return args.Get(0).(istructs.IStateKeyBuilder), args.Error(1)
}
func (m *MockState) MustExist(key istructs.IStateKeyBuilder) (value istructs.IStateValue, err error) {
	args := m.Called(key)
	return args.Get(0).(istructs.IStateValue), args.Error(1)
}
func (m *MockState) MustExistAll(keys []istructs.IStateKeyBuilder, callback istructs.StateValueCallback) (err error) {
	args := m.Called(keys, callback)
	return args.Error(0)
}
func (m *MockState) MustNotExist(key istructs.IStateKeyBuilder) (err error) {
	args := m.Called(key)
	return args.Error(0)
}
func (m *MockState) MustNotExistAll(keys []istructs.IStateKeyBuilder) (err error) {
	args := m.Called(keys)
	return args.Error(0)
}
func (m *MockState) PLogEvent() istructs.IPLogEvent {
	args := m.Called()
	return args.Get(0).(istructs.IPLogEvent)
}
func (m *MockState) QueryPrepareArgs() istructs.PrepareArgs {
	args := m.Called()
	return args.Get(0).(istructs.PrepareArgs)
}
func (m *MockState) QueryCallback() istructs.ExecQueryCallback {
	args := m.Called()
	return args.Get(0).(istructs.ExecQueryCallback)
}
func (m *MockState) Read(key istructs.IStateKeyBuilder, callback istructs.ValueCallback) (err error) {
	args := m.Called(key, callback)
	return args.Error(0)
}

type MockStateKeyBuilder struct {
	mock.Mock
}

func (m *MockStateKeyBuilder) PutInt32(name appdef.FieldName, value int32) {
	m.Called(name, value)
}
func (m *MockStateKeyBuilder) PutInt64(name appdef.FieldName, value int64) {
	m.Called(name, value)
}
func (m *MockStateKeyBuilder) PutFloat32(name appdef.FieldName, value float32) {
	m.Called(name, value)
}
func (m *MockStateKeyBuilder) PutFloat64(name appdef.FieldName, value float64) {
	m.Called(name, value)
}
func (m *MockStateKeyBuilder) PutBytes(name appdef.FieldName, value []byte) {
	m.Called(name, value)
}
func (m *MockStateKeyBuilder) PutString(name appdef.FieldName, value string) {
	m.Called(name, value)
}
func (m *MockStateKeyBuilder) PutQName(name appdef.FieldName, value appdef.QName) {
	m.Called(name, value)
}
func (m *MockStateKeyBuilder) PutBool(name appdef.FieldName, value bool) {
	m.Called(name, value)
}
func (m *MockStateKeyBuilder) PutRecordID(name appdef.FieldName, value istructs.RecordID) {
	m.Called(name, value)
}
func (m *MockStateKeyBuilder) PutNumber(name appdef.FieldName, value json.Number) {
	m.Called(name, value)
}
func (m *MockStateKeyBuilder) PutChars(name appdef.FieldName, value string) {
	m.Called(name, value)
}
func (m *MockStateKeyBuilder) PartitionKey() istructs.IRowWriter {
	args := m.Called()
	return args.Get(0).(istructs.IRowWriter)
}
func (m *MockStateKeyBuilder) ClusteringColumns() istructs.IRowWriter {
	args := m.Called()
	return args.Get(0).(istructs.IRowWriter)
}
func (m *MockStateKeyBuilder) Equals(src istructs.IKeyBuilder) bool {
	args := m.Called(src)
	return args.Bool(0)
}
func (m *MockStateKeyBuilder) Storage() appdef.QName {
	args := m.Called()
	return args.Get(0).(appdef.QName)
}
func (m *MockStateKeyBuilder) Entity() appdef.QName {
	args := m.Called()
	return args.Get(0).(appdef.QName)
}
func (m *MockStateKeyBuilder) PutFromJSON(map[string]any) {
	m.Called(0)
}
func (m *MockStateKeyBuilder) ToBytes(wsid istructs.WSID) ([]byte, []byte, error) {
	args := m.Called(wsid)
	return args.Get(0).([]byte), args.Get(1).([]byte), args.Error(2)
}

type MockStateValue struct {
	mock.Mock
}

func (m *MockStateValue) AsInt32(name appdef.FieldName) int32 { return m.Called(name).Get(0).(int32) }
func (m *MockStateValue) AsInt64(name appdef.FieldName) int64 { return m.Called(name).Get(0).(int64) }
func (m *MockStateValue) AsFloat32(name appdef.FieldName) float32 {
	return m.Called(name).Get(0).(float32)
}
func (m *MockStateValue) AsFloat64(name appdef.FieldName) float64 {
	return m.Called(name).Get(0).(float64)
}
func (m *MockStateValue) AsBytes(name appdef.FieldName) []byte { return m.Called(name).Get(0).([]byte) }
func (m *MockStateValue) AsString(name appdef.FieldName) string {
	return m.Called(name).Get(0).(string)
}
func (m *MockStateValue) AsQName(name appdef.FieldName) appdef.QName {
	return m.Called(name).Get(0).(appdef.QName)
}
func (m *MockStateValue) AsBool(name appdef.FieldName) bool { return m.Called(name).Get(0).(bool) }
func (m *MockStateValue) AsRecordID(name appdef.FieldName) istructs.RecordID {
	return m.Called(name).Get(0).(istructs.RecordID)
}
func (m *MockStateValue) RecordIDs(includeNulls bool) func(func(appdef.FieldName, istructs.RecordID) bool) {
	return m.Called(includeNulls).Get(0).(func(func(appdef.FieldName, istructs.RecordID) bool))
}
func (m *MockStateValue) FieldNames(cb func(appdef.FieldName) bool) { m.Called(cb) }
func (m *MockStateValue) AsRecord(name appdef.FieldName) istructs.IRecord {
	return m.Called(name).Get(0).(istructs.IRecord)
}
func (m *MockStateValue) AsEvent(name appdef.FieldName) istructs.IDbEvent {
	return m.Called(name).Get(0).(istructs.IDbEvent)
}
func (m *MockStateValue) AsValue(name string) istructs.IStateValue {
	return m.Called(name).Get(0).(istructs.IStateValue)
}
func (m *MockStateValue) Length() int                    { return m.Called().Int(0) }
func (m *MockStateValue) GetAsString(index int) string   { return m.Called(index).Get(0).(string) }
func (m *MockStateValue) GetAsBytes(index int) []byte    { return m.Called(index).Get(0).([]byte) }
func (m *MockStateValue) GetAsInt32(index int) int32     { return m.Called(index).Get(0).(int32) }
func (m *MockStateValue) GetAsInt64(index int) int64     { return m.Called(index).Get(0).(int64) }
func (m *MockStateValue) GetAsFloat32(index int) float32 { return m.Called(index).Get(0).(float32) }
func (m *MockStateValue) GetAsFloat64(index int) float64 { return m.Called(index).Get(0).(float64) }
func (m *MockStateValue) GetAsQName(index int) appdef.QName {
	return m.Called(index).Get(0).(appdef.QName)
}
func (m *MockStateValue) GetAsBool(index int) bool { return m.Called(index).Get(0).(bool) }
func (m *MockStateValue) GetAsValue(index int) istructs.IStateValue {
	return m.Called(index).Get(0).(istructs.IStateValue)
}

type MockStateValueBuilder struct {
	mock.Mock
}

func (m *MockStateValueBuilder) Equal(src istructs.IStateValueBuilder) bool {
	return true
}
func (m *MockStateValueBuilder) PutInt32(name appdef.FieldName, value int32) {
	m.Called(name, value)
}
func (m *MockStateValueBuilder) PutInt64(name appdef.FieldName, value int64) {
	m.Called(name, value)
}
func (m *MockStateValueBuilder) PutFloat32(name appdef.FieldName, value float32) {
	m.Called(name, value)
}
func (m *MockStateValueBuilder) PutFloat64(name appdef.FieldName, value float64) {
	m.Called(name, value)
}
func (m *MockStateValueBuilder) PutBytes(name appdef.FieldName, value []byte) {
	m.Called(name, value)
}
func (m *MockStateValueBuilder) PutString(name appdef.FieldName, value string) {
	m.Called(name, value)
}
func (m *MockStateValueBuilder) PutQName(name appdef.FieldName, value appdef.QName) {
	m.Called(name, value)
}
func (m *MockStateValueBuilder) PutBool(name appdef.FieldName, value bool) {
	m.Called(name, value)
}
func (m *MockStateValueBuilder) PutRecordID(name appdef.FieldName, value istructs.RecordID) {
	m.Called(name, value)
}
func (m *MockStateValueBuilder) PutNumber(name appdef.FieldName, value json.Number) {
	m.Called(name, value)
}
func (m *MockStateValueBuilder) PutChars(name appdef.FieldName, value string) {
	m.Called(name, value)
}
func (m *MockStateValueBuilder) PutRecord(name appdef.FieldName, record istructs.IRecord) {
	m.Called(name, record)
}
func (m *MockStateValueBuilder) PutEvent(name appdef.FieldName, event istructs.IDbEvent) {
	m.Called(name, event)
}
func (m *MockStateValueBuilder) Build() istructs.IValue {
	args := m.Called()
	return args.Get(0).(istructs.IValue)
}
func (m *MockStateValueBuilder) BuildValue() istructs.IStateValue {
	args := m.Called()
	return args.Get(0).(istructs.IStateValue)
}
func (m *MockStateValueBuilder) PutFromJSON(map[string]any) {
	m.Called(0)
}
func (m *MockStateValueBuilder) ToBytes() ([]byte, error) {
	args := m.Called()
	return args.Get(0).([]byte), args.Error(1)
}

type MockRawEvent struct {
	mock.Mock
}

func (m *MockRawEvent) QName() appdef.QName                     { return m.Called().Get(0).(appdef.QName) }
func (m *MockRawEvent) ArgumentObject() istructs.IObject        { return m.Called().Get(0).(istructs.IObject) }
func (m *MockRawEvent) CUDs(cb func(rec istructs.ICUDRow) bool) { m.Called(cb) }
func (m *MockRawEvent) SyncedAt() istructs.UnixMilli            { return m.Called().Get(0).(istructs.UnixMilli) }
func (m *MockRawEvent) Synced() bool                            { return m.Called().Bool(0) }
func (m *MockRawEvent) PLogOffset() istructs.Offset             { return m.Called().Get(0).(istructs.Offset) }
func (m *MockRawEvent) Workspace() istructs.WSID                { return m.Called().Get(0).(istructs.WSID) }
func (m *MockRawEvent) WLogOffset() istructs.Offset             { return m.Called().Get(0).(istructs.Offset) }
func (m *MockRawEvent) RegisteredAt() istructs.UnixMilli {
	return m.Called().Get(0).(istructs.UnixMilli)
}
func (m *MockRawEvent) DeviceID() istructs.ConnectedDeviceID {
	return m.Called().Get(0).(istructs.ConnectedDeviceID)
}
func (m *MockRawEvent) ArgumentUnloggedObject() istructs.IObject {
	return m.Called().Get(0).(istructs.IObject)
}
func (m *MockRawEvent) HandlingPartition() istructs.PartitionID {
	return m.Called().Get(0).(istructs.PartitionID)
}

type MockKey struct {
	mock.Mock
}

func (m *MockKey) AsInt32(name appdef.FieldName) int32     { return m.Called(name).Get(0).(int32) }
func (m *MockKey) AsInt64(name appdef.FieldName) int64     { return m.Called(name).Get(0).(int64) }
func (m *MockKey) AsFloat32(name appdef.FieldName) float32 { return m.Called(name).Get(0).(float32) }
func (m *MockKey) AsFloat64(name appdef.FieldName) float64 { return m.Called(name).Get(0).(float64) }
func (m *MockKey) AsBytes(name appdef.FieldName) []byte    { return m.Called(name).Get(0).([]byte) }
func (m *MockKey) AsString(name appdef.FieldName) string   { return m.Called(name).Get(0).(string) }
func (m *MockKey) AsQName(name appdef.FieldName) appdef.QName {
	return m.Called(name).Get(0).(appdef.QName)
}
func (m *MockKey) AsBool(name appdef.FieldName) bool         { return m.Called(name).Get(0).(bool) }
func (m *MockKey) FieldNames(cb func(appdef.FieldName) bool) { m.Called(cb) }
func (m *MockKey) AsRecordID(name appdef.FieldName) istructs.RecordID {
	return m.Called(name).Get(0).(istructs.RecordID)
}
func (m *MockKey) RecordIDs(includeNulls bool) func(func(appdef.FieldName, istructs.RecordID) bool) {
	return m.Called(includeNulls).Get(0).(func(func(appdef.FieldName, istructs.RecordID) bool))
}

type MockIntents struct {
	mock.Mock
}

func (m *MockIntents) FindIntentWithOpKind(key istructs.IStateKeyBuilder) (istructs.IStateValueBuilder, bool) {
	return nil, false
}

func (m *MockIntents) Intents(iterFunc func(key istructs.IStateKeyBuilder, value istructs.IStateValueBuilder, isNew bool)) {
}

func (m *MockIntents) FindIntent(key istructs.IStateKeyBuilder) istructs.IStateValueBuilder {
	return nil
}
func (m *MockIntents) IntentsCount() int {
	return 0
}
func (m *MockIntents) NewValue(key istructs.IStateKeyBuilder) (istructs.IStateValueBuilder, error) {
	args := m.Called(key)
	return args.Get(0).(istructs.IStateValueBuilder), args.Error(1)
}
func (m *MockIntents) UpdateValue(key istructs.IStateKeyBuilder, existingValue istructs.IStateValue) (istructs.IStateValueBuilder, error) {
	args := m.Called(key, existingValue)
	return args.Get(0).(istructs.IStateValueBuilder), args.Error(1)
}
