/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package mock

import (
	"github.com/stretchr/testify/mock"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

type CUDRow struct {
	istructs.ICUDRow
	mock.Mock
}

func (r *CUDRow) AsString(name string) string { return r.Called(name).String(0) }
func (r *CUDRow) AsBool(name string) bool     { return r.Called(name).Bool(0) }
func (r *CUDRow) QName() appdef.QName         { return r.Called().Get(0).(appdef.QName) }
func (r *CUDRow) ID() istructs.RecordID       { return r.Called().Get(0).(istructs.RecordID) }

type PLogEvent struct {
	istructs.IPLogEvent
	mock.Mock
}

func (e *PLogEvent) ArgumentObject() istructs.IObject    { return e.Called().Get(0).(istructs.IObject) }
func (e *PLogEvent) CUDs(cb func(istructs.ICUDRow) bool) { e.Called(cb) }
func (e *PLogEvent) Workspace() istructs.WSID            { return e.Called().Get(0).(istructs.WSID) }

type State struct {
	istructs.IState
	mock.Mock
}

func (s *State) KeyBuilder(storage, entity appdef.QName) (builder istructs.IStateKeyBuilder, err error) {
	args := s.Called(storage, entity)
	if intf := args.Get(0); intf != nil {
		builder = intf.(istructs.IStateKeyBuilder)
	}
	err = args.Error(1)
	return
}
func (s *State) MustExist(key istructs.IStateKeyBuilder) (value istructs.IStateValue, err error) {
	args := s.Called(key)
	if intf := args.Get(0); intf != nil {
		value = intf.(istructs.IStateValue)
	}
	err = args.Error(1)
	return
}

type Intents struct {
	istructs.IIntents
	mock.Mock
}

func (i *Intents) NewValue(key istructs.IStateKeyBuilder) (builder istructs.IStateValueBuilder, err error) {
	args := i.Called(key)
	if intf := args.Get(0); intf != nil {
		builder = intf.(istructs.IStateValueBuilder)
	}
	err = args.Error(1)
	return
}

type StateKeyBuilder struct {
	istructs.IStateKeyBuilder
	mock.Mock
}

func (b *StateKeyBuilder) PutInt32(name string, value int32)        { b.Called(name, value) }
func (b *StateKeyBuilder) PutInt64(name string, value int64)        { b.Called(name, value) }
func (b *StateKeyBuilder) PutString(name, value string)             { b.Called(name, value) }
func (b *StateKeyBuilder) PutQName(name string, value appdef.QName) { b.Called(name, value) }
func (b *StateKeyBuilder) String() string                           { return b.Called().String(0) }

type StateValueBuilder struct {
	istructs.IStateValueBuilder
	mock.Mock
}

type StateValue struct {
	istructs.IStateValue
	mock.Mock
}

func (v *StateValue) AsInt32(name string) int32   { return v.Called(name).Get(0).(int32) }
func (v *StateValue) AsInt64(name string) int64   { return v.Called(name).Get(0).(int64) }
func (v *StateValue) AsString(name string) string { return v.Called(name).String(0) }

type AppStructsProvider struct {
	mock.Mock
}

func (p *AppStructsProvider) AppStructs(aqn appdef.AppQName) (structs istructs.IAppStructs, err error) {
	args := p.Called(aqn)
	if intf := args.Get(0); intf != nil {
		structs = intf.(istructs.IAppStructs)
	}
	err = args.Error(1)
	return
}

type AppStructs struct {
	istructs.IAppStructs
	mock.Mock
}

func (s *AppStructs) Records() istructs.IRecords { return s.Called().Get(0).(istructs.IRecords) }

type Records struct {
	istructs.IRecords
	mock.Mock
}

func (r *Records) GetSingleton(workspace istructs.WSID, qName appdef.QName) (record istructs.IRecord, err error) {
	args := r.Called(workspace, qName)
	if intf := args.Get(0); intf != nil {
		record = intf.(istructs.IRecord)
	}
	err = args.Error(1)
	return
}

type Object struct {
	istructs.IObject
	mock.Mock
}

func (o *Object) AsInt32(name string) int32   { return o.Called(name).Get(0).(int32) }
func (o *Object) AsInt64(name string) int64   { return o.Called(name).Get(0).(int64) }
func (o *Object) AsString(name string) string { return o.Called(name).String(0) }
func (o *Object) Children(container ...string) func(func(istructs.IObject) bool) {
	args := o.Called(container)
	return args.Get(0).(func(func(istructs.IObject) bool))
}

type Record struct {
	istructs.IRecord
	mock.Mock
}

func (r *Record) AsString(name string) string { return r.Called(name).String(0) }
func (r *Record) AsInt64(name string) int64   { return r.Called(name).Get(0).(int64) }
func (r *Record) AsRecordID(name string) istructs.RecordID {
	return r.Called(name).Get(0).(istructs.RecordID)
}
