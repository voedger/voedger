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
