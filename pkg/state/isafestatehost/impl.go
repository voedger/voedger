/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Michael Saigachenko
 */
package isafestatehost

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/state/isafestate"
)

type safeStateHost struct {
	state         state.IUnsafeState
	keyBuilders   []istructs.IStateKeyBuilder
	keys          []istructs.IKey
	values        []istructs.IStateValue
	valueBuilders []istructs.IValueBuilder
}

func (s *safeStateHost) KeyBuilder(storage, entityFull string) isafestate.TKeyBuilder {
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
	kb := isafestate.TKeyBuilder(len(s.keyBuilders))
	s.keyBuilders = append(s.keyBuilders, skb)
	return kb
}

func (s *safeStateHost) MustGetValue(key isafestate.TKeyBuilder) isafestate.TValue {
	sv, err := s.state.MustExist(s.keyBuilders[key])
	if err != nil {
		panic(err)
	}
	v := isafestate.TValue(len(s.values))
	s.values = append(s.values, sv)
	return v
}

func (s *safeStateHost) QueryValue(key isafestate.TKeyBuilder) (isafestate.TValue, bool) {
	sv, ok, err := s.state.CanExist(s.keyBuilders[key])
	if err != nil {
		panic(err)
	}
	if ok {
		v := isafestate.TValue(len(s.values))
		s.values = append(s.values, sv)
		return v, true
	}
	return 0, false
}

func (s *safeStateHost) NewValue(key isafestate.TKeyBuilder) isafestate.TIntent {
	svb, err := s.state.NewValue(s.keyBuilders[key])
	if err != nil {
		panic(err)
	}
	v := isafestate.TIntent(len(s.valueBuilders))
	s.valueBuilders = append(s.valueBuilders, svb)
	return v
}

func (s *safeStateHost) UpdateValue(key isafestate.TKeyBuilder, existingValue isafestate.TValue) isafestate.TIntent {
	svb, err := s.state.UpdateValue(s.keyBuilders[key], s.values[existingValue])
	if err != nil {
		panic(err)
	}
	v := isafestate.TIntent(len(s.valueBuilders))
	s.valueBuilders = append(s.valueBuilders, svb)
	return v
}

func (s *safeStateHost) ReadValues(kb isafestate.TKeyBuilder, callback func(isafestate.TKey, isafestate.TValue)) {
	first := true
	safeKey := isafestate.TKey(len(s.keys))
	safeValue := isafestate.TValue(len(s.values))
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
func (s *safeStateHost) KeyBuilderPutInt32(key isafestate.TKeyBuilder, name string, value int32) {
	s.keyBuilders[key].PutInt32(name, value)
}

func (s *safeStateHost) KeyBuilderPutInt64(key isafestate.TKeyBuilder, name string, value int64) {
	s.keyBuilders[key].PutInt64(name, value)
}

func (s *safeStateHost) KeyBuilderPutFloat32(key isafestate.TKeyBuilder, name string, value float32) {
	s.keyBuilders[key].PutFloat32(name, value)
}

func (s *safeStateHost) KeyBuilderPutFloat64(key isafestate.TKeyBuilder, name string, value float64) {
	s.keyBuilders[key].PutFloat64(name, value)
}

func (s *safeStateHost) KeyBuilderPutString(key isafestate.TKeyBuilder, name string, value string) {
	s.keyBuilders[key].PutString(name, value)
}

func (s *safeStateHost) KeyBuilderPutBytes(key isafestate.TKeyBuilder, name string, value []byte) {
	s.keyBuilders[key].PutBytes(name, value)
}

func (s *safeStateHost) KeyBuilderPutQName(key isafestate.TKeyBuilder, name string, value isafestate.QName) {
	localpkgName := s.state.PackageLocalName(value.FullPkgName)
	s.keyBuilders[key].PutQName(name, appdef.NewQName(localpkgName, value.Entity))
}

func (s *safeStateHost) KeyBuilderPutBool(key isafestate.TKeyBuilder, name string, value bool) {
	s.keyBuilders[key].PutBool(name, value)
}

// Value

func (s *safeStateHost) ValueAsValue(v isafestate.TValue, name string) (result isafestate.TValue) {
	result = isafestate.TValue(len(s.values))
	sv := s.values[v].AsValue(name)
	s.values = append(s.values, sv)
	return result
}

func (s *safeStateHost) ValueAsInt32(v isafestate.TValue, name string) int32 {
	return s.values[v].AsInt32(name)
}

func (s *safeStateHost) ValueAsInt64(v isafestate.TValue, name string) int64 {
	return s.values[v].AsInt64(name)
}

func (s *safeStateHost) ValueAsFloat32(v isafestate.TValue, name string) float32 {
	return s.values[v].AsFloat32(name)
}

func (s *safeStateHost) ValueAsFloat64(v isafestate.TValue, name string) float64 {
	return s.values[v].AsFloat64(name)
}

func (s *safeStateHost) ValueAsBytes(v isafestate.TValue, name string) []byte {
	return s.values[v].AsBytes(name)
}

func (s *safeStateHost) ValueAsQName(v isafestate.TValue, name string) isafestate.QName {
	qname := s.values[v].AsQName(name)
	return isafestate.QName{
		FullPkgName: s.state.PackageFullPath(qname.Pkg()),
		Entity:      qname.Entity(),
	}
}

func (s *safeStateHost) ValueAsBool(v isafestate.TValue, name string) bool {
	return s.values[v].AsBool(name)
}

func (s *safeStateHost) ValueAsString(v isafestate.TValue, name string) string {
	return s.values[v].AsString(name)
}

func (s *safeStateHost) ValueLen(v isafestate.TValue) int {
	return s.values[v].Length()
}

func (s *safeStateHost) ValueGetAsValue(v isafestate.TValue, index int) (result isafestate.TValue) {
	result = isafestate.TValue(len(s.values))
	sv := s.values[v].GetAsValue(index)
	s.values = append(s.values, sv)
	return result
}

func (s *safeStateHost) ValueGetAsInt32(v isafestate.TValue, index int) int32 {
	return s.values[v].GetAsInt32(index)
}

func (s *safeStateHost) ValueGetAsInt64(v isafestate.TValue, index int) int64 {
	return s.values[v].GetAsInt64(index)
}

func (s *safeStateHost) ValueGetAsFloat32(v isafestate.TValue, index int) float32 {
	return s.values[v].GetAsFloat32(index)
}

func (s *safeStateHost) ValueGetAsFloat64(v isafestate.TValue, index int) float64 {
	return s.values[v].GetAsFloat64(index)
}

func (s *safeStateHost) ValueGetAsBytes(v isafestate.TValue, index int) []byte {
	return s.values[v].GetAsBytes(index)
}

func (s *safeStateHost) ValueGetAsQName(v isafestate.TValue, index int) isafestate.QName {
	qname := s.values[v].GetAsQName(index)
	return isafestate.QName{
		FullPkgName: s.state.PackageFullPath(qname.Pkg()),
		Entity:      qname.Entity(),
	}
}

func (s *safeStateHost) ValueGetAsBool(v isafestate.TValue, index int) bool {
	return s.values[v].GetAsBool(index)
}

func (s *safeStateHost) ValueGetAsString(v isafestate.TValue, index int) string {
	return s.values[v].GetAsString(index)
}

// Intent

func (s *safeStateHost) IntentPutInt64(v isafestate.TIntent, name string, value int64) {
	s.valueBuilders[v].PutInt64(name, value)
}

func (s *safeStateHost) IntentPutBool(v isafestate.TIntent, name string, value bool) {
	s.valueBuilders[v].PutBool(name, value)
}

func (s *safeStateHost) IntentPutString(v isafestate.TIntent, name string, value string) {
	s.valueBuilders[v].PutString(name, value)
}

func (s *safeStateHost) IntentPutBytes(v isafestate.TIntent, name string, value []byte) {
	s.valueBuilders[v].PutBytes(name, value)
}

func (s *safeStateHost) IntentPutQName(v isafestate.TIntent, name string, value isafestate.QName) {
	localpkgName := s.state.PackageLocalName(value.FullPkgName)
	s.valueBuilders[v].PutQName(name, appdef.NewQName(localpkgName, value.Entity))
}

func (s *safeStateHost) IntentPutInt32(v isafestate.TIntent, name string, value int32) {
	s.valueBuilders[v].PutInt32(name, value)
}

func (s *safeStateHost) IntentPutFloat32(v isafestate.TIntent, name string, value float32) {
	s.valueBuilders[v].PutFloat32(name, value)
}

func (s *safeStateHost) IntentPutFloat64(v isafestate.TIntent, name string, value float64) {
	s.valueBuilders[v].PutFloat64(name, value)
}

// Key
func (s *safeStateHost) KeyAsInt32(k isafestate.TKey, name string) int32 {
	return s.keys[k].AsInt32(name)
}

func (s *safeStateHost) KeyAsInt64(k isafestate.TKey, name string) int64 {
	return s.keys[k].AsInt64(name)
}

func (s *safeStateHost) KeyAsFloat32(k isafestate.TKey, name string) float32 {
	return s.keys[k].AsFloat32(name)
}

func (s *safeStateHost) KeyAsFloat64(k isafestate.TKey, name string) float64 {
	return s.keys[k].AsFloat64(name)
}

func (s *safeStateHost) KeyAsBytes(k isafestate.TKey, name string) []byte {
	return s.keys[k].AsBytes(name)
}

func (s *safeStateHost) KeyAsString(k isafestate.TKey, name string) string {
	return s.keys[k].AsString(name)
}

func (s *safeStateHost) KeyAsQName(k isafestate.TKey, name string) isafestate.QName {
	qname := s.keys[k].AsQName(name)
	return isafestate.QName{
		FullPkgName: s.state.PackageFullPath(qname.Pkg()),
		Entity:      qname.Entity(),
	}
}

func (s *safeStateHost) KeyAsBool(k isafestate.TKey, name string) bool {
	return s.keys[k].AsBool(name)
}

func provideSafeStateImpl(state state.IUnsafeState) isafestate.ISafeState {
	return &safeStateHost{
		state: state,
	}
}
