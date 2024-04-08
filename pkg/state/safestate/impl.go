/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Michael Saigachenko
 */
package safestate

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/state/isafeapi"
)

type safeState struct {
	state         state.IState
	keyBuilders   []istructs.IStateKeyBuilder
	keys          []istructs.IKey
	values        []istructs.IStateValue
	valueBuilders []istructs.IValueBuilder
}

func (s *safeState) KeyBuilder(storage, entityFull string) isafeapi.TKeyBuilder {
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
	kb := isafeapi.TKeyBuilder(len(s.keyBuilders))
	s.keyBuilders = append(s.keyBuilders, skb)
	return kb
}

func (s *safeState) MustGetValue(key isafeapi.TKeyBuilder) isafeapi.TValue {
	if int(key) >= len(s.keyBuilders) {
		panic(PanicIncorrectKeyBuilder)
	}
	sv, err := s.state.MustExist(s.keyBuilders[key])
	if err != nil {
		panic(err)
	}
	v := isafeapi.TValue(len(s.values))
	s.values = append(s.values, sv)
	return v
}

func (s *safeState) QueryValue(key isafeapi.TKeyBuilder) (isafeapi.TValue, bool) {
	sv, ok, err := s.state.CanExist(s.keyBuilders[key])
	if err != nil {
		panic(err)
	}
	if ok {
		v := isafeapi.TValue(len(s.values))
		s.values = append(s.values, sv)
		return v, true
	}
	return 0, false
}

func (s *safeState) NewValue(key isafeapi.TKeyBuilder) isafeapi.TIntent {
	svb, err := s.state.NewValue(s.keyBuilders[key])
	if err != nil {
		panic(err)
	}
	v := isafeapi.TIntent(len(s.valueBuilders))
	s.valueBuilders = append(s.valueBuilders, svb)
	return v
}

func (s *safeState) UpdateValue(key isafeapi.TKeyBuilder, existingValue isafeapi.TValue) isafeapi.TIntent {
	svb, err := s.state.UpdateValue(s.keyBuilders[key], s.values[existingValue])
	if err != nil {
		panic(err)
	}
	v := isafeapi.TIntent(len(s.valueBuilders))
	s.valueBuilders = append(s.valueBuilders, svb)
	return v
}

func (s *safeState) ReadValues(kb isafeapi.TKeyBuilder, callback func(isafeapi.TKey, isafeapi.TValue)) {
	if int(kb) >= len(s.keyBuilders) {
		panic(PanicIncorrectKeyBuilder)
	}
	first := true
	safeKey := isafeapi.TKey(len(s.keys))
	safeValue := isafeapi.TValue(len(s.values))
	err := s.state.Read(s.keyBuilders[kb], func(key istructs.IKey, value istructs.IStateValue) error {
		if first {
			s.keys = append(s.keys, key)
			s.values = append(s.values, value)
			first = false
		} else { // replace
			s.keys[safeKey] = key
			s.values[safeValue] = value
		}
		callback(safeKey, safeValue)
		return nil
	})
	if err != nil {
		panic(err)
	}
	//TODO: cleanup keys and values
}

// Key Builder
func (s *safeState) KeyBuilderPutInt32(key isafeapi.TKeyBuilder, name string, value int32) {
	s.keyBuilders[key].PutInt32(name, value)
}

func (s *safeState) KeyBuilderPutInt64(key isafeapi.TKeyBuilder, name string, value int64) {
	s.keyBuilders[key].PutInt64(name, value)
}

func (s *safeState) KeyBuilderPutFloat32(key isafeapi.TKeyBuilder, name string, value float32) {
	s.keyBuilders[key].PutFloat32(name, value)
}

func (s *safeState) KeyBuilderPutFloat64(key isafeapi.TKeyBuilder, name string, value float64) {
	s.keyBuilders[key].PutFloat64(name, value)
}

func (s *safeState) KeyBuilderPutString(key isafeapi.TKeyBuilder, name string, value string) {
	s.keyBuilders[key].PutString(name, value)
}

func (s *safeState) KeyBuilderPutBytes(key isafeapi.TKeyBuilder, name string, value []byte) {
	s.keyBuilders[key].PutBytes(name, value)
}

func (s *safeState) KeyBuilderPutQName(key isafeapi.TKeyBuilder, name string, value isafeapi.QName) {
	localpkgName := s.state.PackageLocalName(value.FullPkgName)
	s.keyBuilders[key].PutQName(name, appdef.NewQName(localpkgName, value.Entity))
}

func (s *safeState) KeyBuilderPutBool(key isafeapi.TKeyBuilder, name string, value bool) {
	s.keyBuilders[key].PutBool(name, value)
}

// Value

func (s *safeState) ValueAsValue(v isafeapi.TValue, name string) (result isafeapi.TValue) {
	result = isafeapi.TValue(len(s.values))
	sv := s.values[v].AsValue(name)
	s.values = append(s.values, sv)
	return result
}

func (s *safeState) ValueAsInt32(v isafeapi.TValue, name string) int32 {
	return s.values[v].AsInt32(name)
}

func (s *safeState) ValueAsInt64(v isafeapi.TValue, name string) int64 {
	return s.values[v].AsInt64(name)
}

func (s *safeState) ValueAsFloat32(v isafeapi.TValue, name string) float32 {
	return s.values[v].AsFloat32(name)
}

func (s *safeState) ValueAsFloat64(v isafeapi.TValue, name string) float64 {
	return s.values[v].AsFloat64(name)
}

func (s *safeState) ValueAsBytes(v isafeapi.TValue, name string) []byte {
	return s.values[v].AsBytes(name)
}

func (s *safeState) ValueAsQName(v isafeapi.TValue, name string) isafeapi.QName {
	qname := s.values[v].AsQName(name)
	return isafeapi.QName{
		FullPkgName: s.state.PackageFullPath(qname.Pkg()),
		Entity:      qname.Entity(),
	}
}

func (s *safeState) ValueAsBool(v isafeapi.TValue, name string) bool {
	return s.values[v].AsBool(name)
}

func (s *safeState) ValueAsString(v isafeapi.TValue, name string) string {
	return s.values[v].AsString(name)
}

func (s *safeState) ValueLen(v isafeapi.TValue) int {
	return s.values[v].Length()
}

func (s *safeState) ValueGetAsValue(v isafeapi.TValue, index int) (result isafeapi.TValue) {
	result = isafeapi.TValue(len(s.values))
	sv := s.values[v].GetAsValue(index)
	s.values = append(s.values, sv)
	return result
}

func (s *safeState) ValueGetAsInt32(v isafeapi.TValue, index int) int32 {
	return s.values[v].GetAsInt32(index)
}

func (s *safeState) ValueGetAsInt64(v isafeapi.TValue, index int) int64 {
	return s.values[v].GetAsInt64(index)
}

func (s *safeState) ValueGetAsFloat32(v isafeapi.TValue, index int) float32 {
	return s.values[v].GetAsFloat32(index)
}

func (s *safeState) ValueGetAsFloat64(v isafeapi.TValue, index int) float64 {
	return s.values[v].GetAsFloat64(index)
}

func (s *safeState) ValueGetAsBytes(v isafeapi.TValue, index int) []byte {
	return s.values[v].GetAsBytes(index)
}

func (s *safeState) ValueGetAsQName(v isafeapi.TValue, index int) isafeapi.QName {
	qname := s.values[v].GetAsQName(index)
	return isafeapi.QName{
		FullPkgName: s.state.PackageFullPath(qname.Pkg()),
		Entity:      qname.Entity(),
	}
}

func (s *safeState) ValueGetAsBool(v isafeapi.TValue, index int) bool {
	return s.values[v].GetAsBool(index)
}

func (s *safeState) ValueGetAsString(v isafeapi.TValue, index int) string {
	return s.values[v].GetAsString(index)
}

// Intent

func (s *safeState) IntentPutInt64(v isafeapi.TIntent, name string, value int64) {
	s.valueBuilders[v].PutInt64(name, value)
}

func (s *safeState) IntentPutBool(v isafeapi.TIntent, name string, value bool) {
	s.valueBuilders[v].PutBool(name, value)
}

func (s *safeState) IntentPutString(v isafeapi.TIntent, name string, value string) {
	s.valueBuilders[v].PutString(name, value)
}

func (s *safeState) IntentPutBytes(v isafeapi.TIntent, name string, value []byte) {
	s.valueBuilders[v].PutBytes(name, value)
}

func (s *safeState) IntentPutQName(v isafeapi.TIntent, name string, value isafeapi.QName) {
	localpkgName := s.state.PackageLocalName(value.FullPkgName)
	s.valueBuilders[v].PutQName(name, appdef.NewQName(localpkgName, value.Entity))
}

func (s *safeState) IntentPutInt32(v isafeapi.TIntent, name string, value int32) {
	s.valueBuilders[v].PutInt32(name, value)
}

func (s *safeState) IntentPutFloat32(v isafeapi.TIntent, name string, value float32) {
	s.valueBuilders[v].PutFloat32(name, value)
}

func (s *safeState) IntentPutFloat64(v isafeapi.TIntent, name string, value float64) {
	s.valueBuilders[v].PutFloat64(name, value)
}

// Key
func (s *safeState) KeyAsInt32(k isafeapi.TKey, name string) int32 {
	return s.keys[k].AsInt32(name)
}

func (s *safeState) KeyAsInt64(k isafeapi.TKey, name string) int64 {
	return s.keys[k].AsInt64(name)
}

func (s *safeState) KeyAsFloat32(k isafeapi.TKey, name string) float32 {
	return s.keys[k].AsFloat32(name)
}

func (s *safeState) KeyAsFloat64(k isafeapi.TKey, name string) float64 {
	return s.keys[k].AsFloat64(name)
}

func (s *safeState) KeyAsBytes(k isafeapi.TKey, name string) []byte {
	return s.keys[k].AsBytes(name)
}

func (s *safeState) KeyAsString(k isafeapi.TKey, name string) string {
	return s.keys[k].AsString(name)
}

func (s *safeState) KeyAsQName(k isafeapi.TKey, name string) isafeapi.QName {
	qname := s.keys[k].AsQName(name)
	return isafeapi.QName{
		FullPkgName: s.state.PackageFullPath(qname.Pkg()),
		Entity:      qname.Entity(),
	}
}

func (s *safeState) KeyAsBool(k isafeapi.TKey, name string) bool {
	return s.keys[k].AsBool(name)
}
