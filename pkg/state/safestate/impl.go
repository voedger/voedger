/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Michael Saigachenko
 */
package safestate

import (
	"errors"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
	safe "github.com/voedger/voedger/pkg/state/isafestateapi"
)

type safeState struct {
	state         state.IState
	keyBuilders   []istructs.IStateKeyBuilder
	keys          []istructs.IKey
	values        []istructs.IStateValue
	valueBuilders []istructs.IStateValueBuilder
}

func (s *safeState) KeyBuilder(storage, entityFull string) safe.TKeyBuilder {

	storageQname := appdef.MustParseQName(storage)
	var entityQname appdef.QName
	if entityFull == "" {
		entityQname = appdef.NullQName
	} else {
		entityFullQname := appdef.MustParseFullQName(entityFull)
		entityLocalPkg := s.state.PackageLocalName(entityFullQname.PkgPath())

		if entityLocalPkg == "" {
			panic(errors.New("undefined package: " + entityFullQname.PkgPath()))
		}

		entityQname = appdef.NewQName(entityLocalPkg, entityFullQname.Entity())
	}
	skb, err := s.state.KeyBuilder(storageQname, entityQname)
	if err != nil {
		panic(err)
	}
	kb := safe.TKeyBuilder(len(s.keyBuilders))
	s.keyBuilders = append(s.keyBuilders, skb)
	return kb
}

func (s *safeState) kb(key safe.TKeyBuilder) istructs.IStateKeyBuilder {
	if int(key) >= len(s.keyBuilders) {
		panic(PanicIncorrectKeyBuilder)
	}
	return s.keyBuilders[key]
}

func (s *safeState) MustGetValue(key safe.TKeyBuilder) safe.TValue {
	sv, err := s.state.MustExist(s.kb(key))
	if err != nil {
		panic(err)
	}
	v := safe.TValue(len(s.values))
	s.values = append(s.values, sv)
	return v
}

func (s *safeState) QueryValue(key safe.TKeyBuilder) (safe.TValue, bool) {
	sv, ok, err := s.state.CanExist(s.kb(key))
	if err != nil {
		panic(err)
	}
	if ok {
		v := safe.TValue(len(s.values))
		s.values = append(s.values, sv)
		return v, true
	}
	return 0, false
}

func (s *safeState) NewValue(key safe.TKeyBuilder) safe.TIntent {
	svb, err := s.state.NewValue(s.kb(key))
	if err != nil {
		panic(err)
	}
	v := safe.TIntent(len(s.valueBuilders))
	s.valueBuilders = append(s.valueBuilders, svb)
	return v
}

func (s *safeState) UpdateValue(key safe.TKeyBuilder, existingValue safe.TValue) safe.TIntent {
	svb, err := s.state.UpdateValue(s.kb(key), s.value(existingValue))
	if err != nil {
		panic(err)
	}
	v := safe.TIntent(len(s.valueBuilders))
	s.valueBuilders = append(s.valueBuilders, svb)
	return v
}

func (s *safeState) ReadValues(kb safe.TKeyBuilder, callback func(safe.TKey, safe.TValue)) {
	first := true
	safeKey := safe.TKey(len(s.keys))
	safeValue := safe.TValue(len(s.values))
	err := s.state.Read(s.kb(kb), func(key istructs.IKey, value istructs.IStateValue) error {
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
func (s *safeState) KeyBuilderPutInt32(key safe.TKeyBuilder, name string, value int32) {
	s.kb(key).PutInt32(name, value)
}

func (s *safeState) KeyBuilderPutInt64(key safe.TKeyBuilder, name string, value int64) {
	s.kb(key).PutInt64(name, value)
}

func (s *safeState) KeyBuilderPutRecordID(key safe.TKeyBuilder, name string, value int64) {
	//nolint:gosec
	s.kb(key).PutRecordID(name, istructs.RecordID(value))
}

func (s *safeState) KeyBuilderPutFloat32(key safe.TKeyBuilder, name string, value float32) {
	s.kb(key).PutFloat32(name, value)
}

func (s *safeState) KeyBuilderPutFloat64(key safe.TKeyBuilder, name string, value float64) {
	s.kb(key).PutFloat64(name, value)
}

func (s *safeState) KeyBuilderPutString(key safe.TKeyBuilder, name string, value string) {
	s.kb(key).PutString(name, value)
}

func (s *safeState) KeyBuilderPutBytes(key safe.TKeyBuilder, name string, value []byte) {
	s.kb(key).PutBytes(name, value)
}

func (s *safeState) KeyBuilderPutQName(key safe.TKeyBuilder, name string, value safe.QName) {
	localpkgName := s.state.PackageLocalName(value.FullPkgName)
	s.kb(key).PutQName(name, appdef.NewQName(localpkgName, value.Entity))
}

func (s *safeState) KeyBuilderPutBool(key safe.TKeyBuilder, name string, value bool) {
	s.kb(key).PutBool(name, value)
}

// Value

func (s *safeState) ValueAsValue(v safe.TValue, name string) (result safe.TValue) {
	sv := s.value(v).AsValue(name)
	result = safe.TValue(len(s.values))
	s.values = append(s.values, sv)
	return result
}

func (s *safeState) ValueAsInt32(v safe.TValue, name string) int32 {
	return s.value(v).AsInt32(name)
}

func (s *safeState) ValueAsInt64(v safe.TValue, name string) int64 {
	return s.value(v).AsInt64(name)
}

func (s *safeState) ValueAsFloat32(v safe.TValue, name string) float32 {
	return s.value(v).AsFloat32(name)
}

func (s *safeState) ValueAsFloat64(v safe.TValue, name string) float64 {
	return s.value(v).AsFloat64(name)
}

func (s *safeState) ValueAsBytes(v safe.TValue, name string) []byte {
	return s.value(v).AsBytes(name)
}

func (s *safeState) ValueAsQName(v safe.TValue, name string) safe.QName {
	qname := s.value(v).AsQName(name)
	return safe.QName{
		FullPkgName: s.state.PackageFullPath(qname.Pkg()),
		Entity:      qname.Entity(),
	}
}

func (s *safeState) ValueAsBool(v safe.TValue, name string) bool {
	return s.value(v).AsBool(name)
}

func (s *safeState) ValueAsString(v safe.TValue, name string) string {
	return s.value(v).AsString(name)
}

func (s *safeState) ValueLen(v safe.TValue) int {
	return s.value(v).Length()
}

func (s *safeState) ValueGetAsValue(v safe.TValue, index int) (result safe.TValue) {
	sv := s.value(v).GetAsValue(index)
	result = safe.TValue(len(s.values))
	s.values = append(s.values, sv)
	return result
}

func (s *safeState) ValueGetAsInt32(v safe.TValue, index int) int32 {
	return s.value(v).GetAsInt32(index)
}

func (s *safeState) ValueGetAsInt64(v safe.TValue, index int) int64 {
	return s.value(v).GetAsInt64(index)
}

func (s *safeState) ValueGetAsFloat32(v safe.TValue, index int) float32 {
	return s.value(v).GetAsFloat32(index)
}

func (s *safeState) ValueGetAsFloat64(v safe.TValue, index int) float64 {
	return s.value(v).GetAsFloat64(index)
}

func (s *safeState) ValueGetAsBytes(v safe.TValue, index int) []byte {
	return s.value(v).GetAsBytes(index)
}

func (s *safeState) ValueGetAsQName(v safe.TValue, index int) safe.QName {
	qname := s.value(v).GetAsQName(index)
	return safe.QName{
		FullPkgName: s.state.PackageFullPath(qname.Pkg()),
		Entity:      qname.Entity(),
	}
}

func (s *safeState) ValueGetAsBool(v safe.TValue, index int) bool {
	return s.value(v).GetAsBool(index)
}

func (s *safeState) ValueGetAsString(v safe.TValue, index int) string {
	return s.value(v).GetAsString(index)
}

func (s *safeState) value(v safe.TValue) istructs.IStateValue {
	if int(v) >= len(s.values) {
		panic(PanicIncorrectValue)
	}
	return s.values[v]
}

// Intent

func (s *safeState) vb(v safe.TIntent) istructs.IStateValueBuilder {
	if int(v) >= len(s.valueBuilders) {
		panic(PanicIncorrectIntent)
	}
	return s.valueBuilders[v]
}

func (s *safeState) IntentPutInt64(v safe.TIntent, name string, value int64) {
	s.vb(v).PutInt64(name, value)
}

func (s *safeState) IntentPutBool(v safe.TIntent, name string, value bool) {
	s.vb(v).PutBool(name, value)
}

func (s *safeState) IntentPutString(v safe.TIntent, name string, value string) {
	s.vb(v).PutString(name, value)
}

func (s *safeState) IntentPutBytes(v safe.TIntent, name string, value []byte) {
	s.vb(v).PutBytes(name, value)
}

func (s *safeState) IntentPutQName(v safe.TIntent, name string, value safe.QName) {
	localpkgName := s.state.PackageLocalName(value.FullPkgName)
	s.vb(v).PutQName(name, appdef.NewQName(localpkgName, value.Entity))
}

func (s *safeState) IntentPutInt32(v safe.TIntent, name string, value int32) {
	s.vb(v).PutInt32(name, value)
}

func (s *safeState) IntentPutFloat32(v safe.TIntent, name string, value float32) {
	s.vb(v).PutFloat32(name, value)
}

func (s *safeState) IntentPutFloat64(v safe.TIntent, name string, value float64) {
	s.vb(v).PutFloat64(name, value)
}

// Key

func (s *safeState) key(k safe.TKey) istructs.IKey {
	if int(k) >= len(s.keys) {
		panic(PanicIncorrectKey)
	}
	return s.keys[k]
}

func (s *safeState) KeyAsInt32(k safe.TKey, name string) int32 {
	return s.key(k).AsInt32(name)
}

func (s *safeState) KeyAsInt64(k safe.TKey, name string) int64 {
	return s.key(k).AsInt64(name)
}

func (s *safeState) KeyAsFloat32(k safe.TKey, name string) float32 {
	return s.key(k).AsFloat32(name)
}

func (s *safeState) KeyAsFloat64(k safe.TKey, name string) float64 {
	return s.key(k).AsFloat64(name)
}

func (s *safeState) KeyAsBytes(k safe.TKey, name string) []byte {
	return s.key(k).AsBytes(name)
}

func (s *safeState) KeyAsString(k safe.TKey, name string) string {
	return s.key(k).AsString(name)
}

func (s *safeState) KeyAsQName(k safe.TKey, name string) safe.QName {
	qname := s.key(k).AsQName(name)
	return safe.QName{
		FullPkgName: s.state.PackageFullPath(qname.Pkg()),
		Entity:      qname.Entity(),
	}
}

func (s *safeState) KeyAsBool(k safe.TKey, name string) bool {
	return s.key(k).AsBool(name)
}
