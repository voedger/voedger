/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Michael Saigachenko
 */
package safestate

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
)

type safeState struct {
	state         state.IUnsafeState
	keyBuilders   []istructs.IStateKeyBuilder
	values        []istructs.IStateValue
	valueBuilders []istructs.IValueBuilder
}

func (s *safeState) KeyBuilder(storage, entityFull string) TSafeKeyBuilder {
	storageQname := appdef.MustParseQName(storage)
	var entityQname appdef.QName
	if entityFull == "" {
		entityQname = appdef.NullQName
	} else {
		entityFullQname := appdef.MustParseFullQName(entityFull)
		entityLocalPkg := s.state.PackageLocalName(entityFullQname.PkgPath())
		entityQname = appdef.NewQName(entityLocalPkg, entityFullQname.Entity())
	}
	skb, err := s.state.KeyBuilder(storageQname, entityQname)
	if err != nil {
		panic(err)
	}
	kb := TSafeKeyBuilder(len(s.keyBuilders))
	s.keyBuilders = append(s.keyBuilders, skb)
	return kb
}

func (s *safeState) MustGetValue(key TSafeKeyBuilder) TSafeValue {
	sv, err := s.state.MustExist(s.keyBuilders[key])
	if err != nil {
		panic(err)
	}
	v := TSafeValue(len(s.values))
	s.values = append(s.values, sv)
	return v
}

func (s *safeState) QueryValue(key TSafeKeyBuilder) (TSafeValue, bool) {
	sv, ok, err := s.state.CanExist(s.keyBuilders[key])
	if err != nil {
		panic(err)
	}
	if ok {
		v := TSafeValue(len(s.values))
		s.values = append(s.values, sv)
		return v, true
	}
	return 0, false
}

func (s *safeState) NewValue(key TSafeKeyBuilder) TSafeIntent {
	svb, err := s.state.NewValue(s.keyBuilders[key])
	if err != nil {
		panic(err)
	}
	v := TSafeIntent(len(s.valueBuilders))
	s.valueBuilders = append(s.valueBuilders, svb)
	return v
}

func (s *safeState) UpdateValue(key TSafeKeyBuilder, existingValue TSafeValue) TSafeIntent {
	svb, err := s.state.UpdateValue(s.keyBuilders[key], s.values[existingValue])
	if err != nil {
		panic(err)
	}
	v := TSafeIntent(len(s.valueBuilders))
	s.valueBuilders = append(s.valueBuilders, svb)
	return v
}

func (s *safeState) KeyBuilderPutInt32(key TSafeKeyBuilder, name string, value int32) {
	s.keyBuilders[key].PutInt32(name, value)
}

func (s *safeState) ValueAsValue(v TSafeValue, name string) (result TSafeValue) {
	result = TSafeValue(len(s.values))
	sv := s.values[v].AsValue(name)
	s.values = append(s.values, sv)
	return result
}

func (s *safeState) ValueLen(v TSafeValue) int {
	return s.values[v].Length()
}

func (s *safeState) ValueGetAsValue(v TSafeValue, index int) (result TSafeValue) {
	result = TSafeValue(len(s.values))
	sv := s.values[v].GetAsValue(index)
	s.values = append(s.values, sv)
	return result
}

func (s *safeState) ValueAsInt32(v TSafeValue, name string) int32 {
	return s.values[v].AsInt32(name)
}

func (s *safeState) ValueAsInt64(v TSafeValue, name string) int64 {
	return s.values[v].AsInt64(name)
}

func (s *safeState) IntentPutInt64(v TSafeIntent, name string, value int64) {
	s.valueBuilders[v].PutInt64(name, value)
}

func provideSafeStateImpl(state state.IUnsafeState) ISafeState {
	return &safeState{
		state: state,
	}
}
