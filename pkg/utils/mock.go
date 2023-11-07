/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 *
 * @author Daniil Solovyov
 */

// This is AI generated code do not edit it manually

package coreutils

import (
	"github.com/stretchr/testify/mock"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

type MockCUDRow struct {
	mock.Mock
}

func (m *MockCUDRow) AsInt32(name string) int32                                      { return m.Called(name).Get(0).(int32) }
func (m *MockCUDRow) AsInt64(name string) int64                                      { return m.Called(name).Get(0).(int64) }
func (m *MockCUDRow) AsFloat32(name string) float32                                  { return m.Called(name).Get(0).(float32) }
func (m *MockCUDRow) AsFloat64(name string) float64                                  { return m.Called(name).Get(0).(float64) }
func (m *MockCUDRow) AsBytes(name string) []byte                                     { return m.Called(name).Get(0).([]byte) }
func (m *MockCUDRow) AsString(name string) string                                    { return m.Called(name).Get(0).(string) }
func (m *MockCUDRow) AsQName(name string) appdef.QName                               { return m.Called(name).Get(0).(appdef.QName) }
func (m *MockCUDRow) AsBool(name string) bool                                        { return m.Called(name).Get(0).(bool) }
func (m *MockCUDRow) FieldNames(cb func(fieldName string))                           { m.Called(cb) }
func (m *MockCUDRow) IsNew() bool                                                    { return m.Called().Get(0).(bool) }
func (m *MockCUDRow) QName() appdef.QName                                            { return m.Called().Get(0).(appdef.QName) }
func (m *MockCUDRow) ID() istructs.RecordID                                          { return m.Called().Get(0).(istructs.RecordID) }
func (m *MockCUDRow) ModifiedFields(cb func(fieldName string, newValue interface{})) { m.Called(cb) }
func (m *MockCUDRow) AsRecordID(name string) istructs.RecordID {
	return m.Called(name).Get(0).(istructs.RecordID)
}
func (m *MockCUDRow) RecordIDs(includeNulls bool, cb func(name string, value istructs.RecordID)) {
	m.Called(includeNulls, cb)
}

type MockPLogEvent struct {
	mock.Mock
}

func (m *MockPLogEvent) ArgumentObject() istructs.IObject {
	return m.Called().Get(0).(istructs.IObject)
}
func (m *MockPLogEvent) CUDs(cb func(rec istructs.ICUDRow) error) (err error) {
	return m.Called(cb).Error(0)
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

func (m *MockObject) AsInt32(name string) int32            { return m.Called(name).Get(0).(int32) }
func (m *MockObject) AsInt64(name string) int64            { return m.Called(name).Get(0).(int64) }
func (m *MockObject) AsFloat32(name string) float32        { return m.Called(name).Get(0).(float32) }
func (m *MockObject) AsFloat64(name string) float64        { return m.Called(name).Get(0).(float64) }
func (m *MockObject) AsBytes(name string) []byte           { return m.Called(name).Get(0).([]byte) }
func (m *MockObject) AsString(name string) string          { return m.Called(name).Get(0).(string) }
func (m *MockObject) AsQName(name string) appdef.QName     { return m.Called(name).Get(0).(appdef.QName) }
func (m *MockObject) AsBool(name string) bool              { return m.Called(name).Get(0).(bool) }
func (m *MockObject) QName() appdef.QName                  { return m.Called().Get(0).(appdef.QName) }
func (m *MockObject) AsRecord() istructs.IRecord           { return m.Called().Get(0).(istructs.IRecord) }
func (m *MockObject) Containers(cb func(container string)) { m.Called(cb) }
func (m *MockObject) FieldNames(cb func(fieldName string)) { m.Called(cb) }
func (m *MockObject) AsRecordID(name string) istructs.RecordID {
	return m.Called(name).Get(0).(istructs.RecordID)
}
func (m *MockObject) RecordIDs(includeNulls bool, cb func(name string, value istructs.RecordID)) {
	m.Called(includeNulls, cb)
}
func (m *MockObject) Elements(container string, cb func(el istructs.IElement)) {
	m.Called(container, cb)
}

type MockState struct {
	mock.Mock
}

func (m *MockState) KeyBuilder(storage, entity appdef.QName) (builder istructs.IStateKeyBuilder, err error) {
	args := m.Called(storage, entity)
	return args.Get(0).(istructs.IStateKeyBuilder), args.Error(1)
}
func (m *MockState) CanExist(key istructs.IStateKeyBuilder) (value istructs.IStateValue, ok bool, err error) {
	args := m.Called(key)
	return args.Get(0).(istructs.IStateValue), args.Bool(1), args.Error(2)
}
func (m *MockState) CanExistAll(keys []istructs.IStateKeyBuilder, callback istructs.StateValueCallback) (err error) {
	args := m.Called(keys, callback)
	return args.Error(0)
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
func (m *MockState) Read(key istructs.IStateKeyBuilder, callback istructs.ValueCallback) (err error) {
	args := m.Called(key, callback)
	return args.Error(0)
}

type MockStateKeyBuilder struct {
	mock.Mock
}

func (m *MockStateKeyBuilder) PutInt32(name string, value int32) {
	m.Called(name, value)
}
func (m *MockStateKeyBuilder) PutInt64(name string, value int64) {
	m.Called(name, value)
}
func (m *MockStateKeyBuilder) PutFloat32(name string, value float32) {
	m.Called(name, value)
}
func (m *MockStateKeyBuilder) PutFloat64(name string, value float64) {
	m.Called(name, value)
}
func (m *MockStateKeyBuilder) PutBytes(name string, value []byte) {
	m.Called(name, value)
}
func (m *MockStateKeyBuilder) PutString(name string, value string) {
	m.Called(name, value)
}
func (m *MockStateKeyBuilder) PutQName(name string, value appdef.QName) {
	m.Called(name, value)
}
func (m *MockStateKeyBuilder) PutBool(name string, value bool) {
	m.Called(name, value)
}
func (m *MockStateKeyBuilder) PutRecordID(name string, value istructs.RecordID) {
	m.Called(name, value)
}
func (m *MockStateKeyBuilder) PutNumber(name string, value float64) {
	m.Called(name, value)
}
func (m *MockStateKeyBuilder) PutChars(name string, value string) {
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

type MockStateValue struct {
	mock.Mock
}

func (m *MockStateValue) AsInt32(name string) int32 {
	args := m.Called(name)
	return args.Get(0).(int32)
}
func (m *MockStateValue) AsInt64(name string) int64 {
	args := m.Called(name)
	return args.Get(0).(int64)
}
func (m *MockStateValue) AsFloat32(name string) float32 {
	args := m.Called(name)
	return args.Get(0).(float32)
}
func (m *MockStateValue) AsFloat64(name string) float64 {
	args := m.Called(name)
	return args.Get(0).(float64)
}
func (m *MockStateValue) AsBytes(name string) []byte {
	args := m.Called(name)
	return args.Get(0).([]byte)
}
func (m *MockStateValue) AsString(name string) string {
	args := m.Called(name)
	return args.Get(0).(string)
}
func (m *MockStateValue) AsQName(name string) appdef.QName {
	args := m.Called(name)
	return args.Get(0).(appdef.QName)
}
func (m *MockStateValue) AsBool(name string) bool {
	args := m.Called(name)
	return args.Get(0).(bool)
}
func (m *MockStateValue) AsRecordID(name string) istructs.RecordID {
	args := m.Called(name)
	return args.Get(0).(istructs.RecordID)
}
func (m *MockStateValue) RecordIDs(includeNulls bool, cb func(name string, value istructs.RecordID)) {
	m.Called(includeNulls, cb)
}
func (m *MockStateValue) FieldNames(cb func(fieldName string)) {
	m.Called(cb)
}
func (m *MockStateValue) AsRecord(name string) istructs.IRecord {
	args := m.Called(name)
	return args.Get(0).(istructs.IRecord)
}
func (m *MockStateValue) AsEvent(name string) istructs.IDbEvent {
	args := m.Called(name)
	return args.Get(0).(istructs.IDbEvent)
}
func (m *MockStateValue) AsValue(name string) istructs.IStateValue {
	args := m.Called(name)
	return args.Get(0).(istructs.IStateValue)
}
func (m *MockStateValue) Length() int {
	args := m.Called()
	return args.Int(0)
}
func (m *MockStateValue) GetAsString(index int) string {
	args := m.Called(index)
	return args.Get(0).(string)
}
func (m *MockStateValue) GetAsBytes(index int) []byte {
	args := m.Called(index)
	return args.Get(0).([]byte)
}
func (m *MockStateValue) GetAsInt32(index int) int32 {
	args := m.Called(index)
	return args.Get(0).(int32)
}
func (m *MockStateValue) GetAsInt64(index int) int64 {
	args := m.Called(index)
	return args.Get(0).(int64)
}
func (m *MockStateValue) GetAsFloat32(index int) float32 {
	args := m.Called(index)
	return args.Get(0).(float32)
}
func (m *MockStateValue) GetAsFloat64(index int) float64 {
	args := m.Called(index)
	return args.Get(0).(float64)
}
func (m *MockStateValue) GetAsQName(index int) appdef.QName {
	args := m.Called(index)
	return args.Get(0).(appdef.QName)
}
func (m *MockStateValue) GetAsBool(index int) bool {
	args := m.Called(index)
	return args.Get(0).(bool)
}
func (m *MockStateValue) GetAsValue(index int) istructs.IStateValue {
	args := m.Called(index)
	return args.Get(0).(istructs.IStateValue)
}

type MockStateValueBuilder struct {
	mock.Mock
}

func (m *MockStateValueBuilder) PutInt32(name string, value int32) {
	m.Called(name, value)
}
func (m *MockStateValueBuilder) PutInt64(name string, value int64) {
	m.Called(name, value)
}
func (m *MockStateValueBuilder) PutFloat32(name string, value float32) {
	m.Called(name, value)
}
func (m *MockStateValueBuilder) PutFloat64(name string, value float64) {
	m.Called(name, value)
}
func (m *MockStateValueBuilder) PutBytes(name string, value []byte) {
	m.Called(name, value)
}
func (m *MockStateValueBuilder) PutString(name string, value string) {
	m.Called(name, value)
}
func (m *MockStateValueBuilder) PutQName(name string, value appdef.QName) {
	m.Called(name, value)
}
func (m *MockStateValueBuilder) PutBool(name string, value bool) {
	m.Called(name, value)
}
func (m *MockStateValueBuilder) PutRecordID(name string, value istructs.RecordID) {
	m.Called(name, value)
}
func (m *MockStateValueBuilder) PutNumber(name string, value float64) {
	m.Called(name, value)
}
func (m *MockStateValueBuilder) PutChars(name string, value string) {
	m.Called(name, value)
}
func (m *MockStateValueBuilder) PutRecord(name string, record istructs.IRecord) {
	m.Called(name, record)
}
func (m *MockStateValueBuilder) PutEvent(name string, event istructs.IDbEvent) {
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

type MockRawEvent struct {
	mock.Mock
}

func (m *MockRawEvent) QName() appdef.QName                            { return m.Called().Get(0).(appdef.QName) }
func (m *MockRawEvent) ArgumentObject() istructs.IObject               { return m.Called().Get(0).(istructs.IObject) }
func (m *MockRawEvent) CUDs(cb func(rec istructs.ICUDRow) error) error { return m.Called(cb).Error(0) }
func (m *MockRawEvent) SyncedAt() istructs.UnixMilli                   { return m.Called().Get(0).(istructs.UnixMilli) }
func (m *MockRawEvent) Synced() bool                                   { return m.Called().Bool(0) }
func (m *MockRawEvent) PLogOffset() istructs.Offset                    { return m.Called().Get(0).(istructs.Offset) }
func (m *MockRawEvent) Workspace() istructs.WSID                       { return m.Called().Get(0).(istructs.WSID) }
func (m *MockRawEvent) WLogOffset() istructs.Offset                    { return m.Called().Get(0).(istructs.Offset) }
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
