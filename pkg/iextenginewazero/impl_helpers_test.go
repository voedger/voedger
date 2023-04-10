/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */

package iextenginewasm

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"

	"github.com/untillpro/voedger/pkg/istructs"
	"github.com/untillpro/voedger/pkg/state"
)

var errTestIOError = errors.New("test i/o error")

var storageEvent = istructs.NewQName("sys", "EventStorage")
var storageSendmail = istructs.NewQName("sys", "SendMailStorage")
var storageRecords = istructs.NewQName("sys", "RecordsStorage")
var storageTest = istructs.NewQName("sys", "Test")
var storageTest2 = istructs.NewQName("sys", "Test2")
var storageTest3 = istructs.NewQName("sys", "Test3")
var storageIoError = istructs.NewQName("sys", "IoErrorStorage")

var projectorMode bool

type mockIo struct {
	istructs.IState
	istructs.IIntents
	intents []intent
}

func testModuleURL(path string) (u *url.URL) {

	path, err := filepath.Abs(path)
	if err != nil {
		panic(err)
	}

	u, err = url.Parse("file:///" + filepath.ToSlash(path))
	if err != nil {
		panic(err)
	}

	return

}

func (s *mockIo) KeyBuilder(storage, entity istructs.QName) (builder istructs.IStateKeyBuilder, err error) {
	return &mockKeyBuilder{
		entity:  entity,
		storage: storage,
	}, nil
}

func mockedValue(name string, value interface{}) istructs.IStateValue {
	mv := mockValue{
		values: make(map[string]interface{}),
	}
	mv.values[name] = value
	return &mv
}

func (s *mockIo) CanExist(key istructs.IStateKeyBuilder) (value istructs.IStateValue, ok bool, err error) {
	k := key.(*mockKeyBuilder)
	mv := mockValue{
		values: make(map[string]interface{}),
	}
	if k.storage == storageIoError {
		return nil, false, errTestIOError
	}
	if k.storage == storageEvent {
		if projectorMode {
			mv.values["offs"] = int32(12345)
			mv.values["qname"] = "air.UpdateSubscription"
			mv.values["arg"] = newJsonValue(`
				{
					"subscription": {
						"status": "active"
					},
					"customer": {
						"email": "customer@test.com"
					}
				}
			`)
			return &mv, true, nil

		}
		mv.values["qname"] = "sys.InvitationAccepted"
		mv.values["arg"] = mockedValue("UserEmail", "email@user.com")
		mv.values["offs"] = int32(12345)
		return &mv, true, nil
	}
	if k.storage == storageTest {
		mv.values["500c"] = "01234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789"
		mv.values["bytes"] = make([]byte, WasmPreallocatedBufferSize*2)
		return &mv, true, nil
	}
	if k.storage == storageTest3 {
		mv.index = make([]interface{}, 4)
		mv.index[0] = int32(123)
		mv.index[1] = "test string"
		mv.index[2] = make([]byte, 1024)
		return &mv, true, nil
	}
	if k.storage == storageTest2 {
		const vvv = "012345678901234567890"
		mv.values["а10"] = vvv
		mv.values["а11"] = vvv
		mv.values["а12"] = vvv
		mv.values["а13"] = vvv
		mv.values["а14"] = vvv
		mv.values["а15"] = vvv
		mv.values["а16"] = vvv
		mv.values["а17"] = vvv
		mv.values["а18"] = vvv
		mv.values["а19"] = vvv
		mv.values["а20"] = vvv
		mv.values["а21"] = vvv
		mv.values["а22"] = vvv
		mv.values["а23"] = vvv
		mv.values["а24"] = vvv
		mv.values["а25"] = vvv
		mv.values["а26"] = vvv
		mv.values["а27"] = vvv
		mv.values["а28"] = vvv
		mv.values["а29"] = vvv
		mv.values["а30"] = vvv
		mv.values["а31"] = vvv
		mv.values["а32"] = vvv
		mv.values["а33"] = vvv
		mv.values["а34"] = vvv
		mv.values["а35"] = vvv
		mv.values["а36"] = vvv
		mv.values["а37"] = vvv
		mv.values["а38"] = vvv
		mv.values["а39"] = vvv
		return &mv, true, nil
	}
	if k.storage == storageSendmail {
		return &mv, true, nil
	}
	if k.storage == storageRecords {
		return &mv, false, nil
	}
	return nil, false, errors.New("unsupported storage: " + k.storage.Pkg() + "." + k.storage.Entity())
}

func (s *mockIo) CanExistAll(keys []istructs.IStateKeyBuilder, callback istructs.StateValueCallback) (err error) {
	return nil
}

func (s *mockIo) MustExist(key istructs.IStateKeyBuilder) (value istructs.IStateValue, err error) {
	k := key.(*mockKeyBuilder)
	if k.storage == storageIoError {
		return nil, errTestIOError
	}
	if k.storage == storageRecords {
		return nil, state.ErrNotExists
	}
	v, ok, err := s.CanExist(key)
	if err != nil {
		return v, err
	}
	if !ok {
		panic("not exists")
	}

	return v, nil
}

func (s *mockIo) MustExistAll(keys []istructs.IStateKeyBuilder, callback istructs.StateValueCallback) (err error) {
	return nil
}

func (s *mockIo) MustNotExist(key istructs.IStateKeyBuilder) (err error) {
	return nil
}

func (s *mockIo) MustNotExistAll(keys []istructs.IStateKeyBuilder) (err error) {
	return nil
}

func (s *mockIo) Read(key istructs.IStateKeyBuilder, callback istructs.ValueCallback) (err error) {
	k := key.(*mockKeyBuilder)
	if k.storage == storageIoError {
		return errTestIOError
	}
	if k.storage == storageTest {
		for i := 1; i <= 3; i++ {
			mk := mockKey{
				values: make(map[string]interface{}),
			}
			mk.values["i32"] = int32(i)
			mk.values["i64"] = 10 + int64(i)
			mk.values["f32"] = float32(i) + 0.1
			mk.values["f64"] = float64(i) + 0.01
			mk.values["str"] = fmt.Sprintf("key%d", i)
			mk.values["bytes"] = []byte{byte(i), 2, 3}
			mk.values["qname"] = istructs.NewQName("keypkg", fmt.Sprintf("e%d", i))
			mk.values["bool"] = true

			mv := mockValue{
				values: make(map[string]interface{}),
			}
			mv.values["i32"] = 100 + int32(i)
			mv.values["i64"] = 1000 + int64(i)
			mv.values["f32"] = float32(i) + 0.001
			mv.values["f64"] = float64(i) + 0.0001
			mv.values["str"] = fmt.Sprintf("value%d", i)
			mv.values["bytes"] = []byte{3, 2, 1}
			mv.values["qname"] = istructs.NewQName("valuepkg", fmt.Sprintf("ee%d", i))
			mv.values["bool"] = false
			if err := callback(&mk, &mv); err != nil {
				return err
			}
		}

	}
	return nil
}

type mockKeyBuilder struct {
	entity  istructs.QName
	storage istructs.QName
}

func (kb *mockKeyBuilder) Entity() istructs.QName                           { return kb.entity }
func (kb *mockKeyBuilder) PartitionKey() istructs.IRowWriter                { return nil }
func (kb *mockKeyBuilder) ClusteringColumns() istructs.IRowWriter           { return nil }
func (kb *mockKeyBuilder) Equals(src istructs.IKeyBuilder) bool             { return false }
func (kb *mockKeyBuilder) PutInt32(name string, value int32)                {}
func (kb *mockKeyBuilder) PutInt64(name string, value int64)                {}
func (kb *mockKeyBuilder) PutFloat32(name string, value float32)            {}
func (kb *mockKeyBuilder) PutFloat64(name string, value float64)            {}
func (kb *mockKeyBuilder) PutBytes(name string, value []byte)               {}
func (kb *mockKeyBuilder) PutString(name, value string)                     {}
func (kb *mockKeyBuilder) PutQName(name string, value istructs.QName)       {}
func (kb *mockKeyBuilder) PutBool(name string, value bool)                  {}
func (kb *mockKeyBuilder) PutRecordID(name string, value istructs.RecordID) {}

// Tries to make conversion from value to a name type
func (kb *mockKeyBuilder) PutNumber(name string, value float64) {}

// Tries to make conversion from value to a name type
func (kb *mockKeyBuilder) PutChars(name string, value string) {}

func newJsonValue(jsonString string) istructs.IStateValue {
	v := mockValue{}
	err := json.Unmarshal([]byte(jsonString), &v.values)
	if err != nil {
		panic(err)
	}
	return &v
}

type mockKey struct {
	istructs.IKey
	values map[string]interface{}
}

func (k *mockKey) AsInt32(name string) int32          { return k.values[name].(int32) }
func (k *mockKey) AsInt64(name string) int64          { return k.values[name].(int64) }
func (k *mockKey) AsFloat32(name string) float32      { return k.values[name].(float32) }
func (k *mockKey) AsFloat64(name string) float64      { return k.values[name].(float64) }
func (k *mockKey) AsBytes(name string) []byte         { return k.values[name].([]byte) }
func (k *mockKey) AsString(name string) string        { return k.values[name].(string) }
func (k *mockKey) AsQName(name string) istructs.QName { return k.values[name].(istructs.QName) }
func (k *mockKey) AsBool(name string) bool            { return k.values[name].(bool) }
func (k *mockKey) AsRecordID(name string) istructs.RecordID {
	return k.values[name].(istructs.RecordID)
}
func (k *mockKey) RecordIDs(includeNulls bool, cb func(name string, value istructs.RecordID)) {}
func (k *mockKey) FieldNames(cb func(fieldName string))                                       {}

type mockValue struct {
	istructs.IStateValue
	values map[string]interface{}
	index  []interface{}
}

func (v *mockValue) ToJSON(opts ...interface{}) (string, error)     { return "", nil }
func (v *mockValue) AsRecord(name string) (record istructs.IRecord) { return nil }
func (v *mockValue) AsEvent(name string) (event istructs.IDbEvent)  { return nil }

func (v *mockValue) AsInt32(name string) int32          { return v.values[name].(int32) }
func (v *mockValue) AsInt64(name string) int64          { return 0 }
func (v *mockValue) AsFloat32(name string) float32      { return 0 }
func (v *mockValue) AsFloat64(name string) float64      { return 0 }
func (v *mockValue) AsBytes(name string) []byte         { return v.values[name].([]byte) }
func (v *mockValue) AsString(name string) string        { return v.values[name].(string) }
func (v *mockValue) AsQName(name string) istructs.QName { return istructs.NullQName }
func (v *mockValue) AsBool(name string) bool            { return false }

func (v *mockValue) GetAsInt32(index int) int32          { return v.index[index].(int32) }
func (v *mockValue) GetAsInt64(index int) int64          { return 0 }
func (v *mockValue) GetAsFloat32(index int) float32      { return 0 }
func (v *mockValue) GetAsFloat64(index int) float64      { return 0 }
func (v *mockValue) GetAsBytes(index int) []byte         { return v.index[index].([]byte) }
func (v *mockValue) GetAsString(index int) string        { return v.index[index].(string) }
func (v *mockValue) GetAsQName(index int) istructs.QName { return istructs.NullQName }
func (v *mockValue) GetAsBool(index int) bool            { return false }

func (v *mockValue) Length() int                              { return 0 }
func (v *mockValue) AsRecordID(name string) istructs.RecordID { return 0 }
func (v *mockValue) GetAsValue(index int) istructs.IStateValue {
	iv, ok := v.index[index].(istructs.IStateValue)
	if ok {
		return iv
	}
	mv, ok := v.index[index].([]interface{})
	if ok {
		return &mockValue{
			index: mv,
		}
	}
	panic(fmt.Sprintf("unsupported value stored under index: %d", index))
}
func (v *mockValue) AsValue(name string) istructs.IStateValue {
	iv, ok := v.values[name].(istructs.IStateValue)
	if ok {
		return iv
	}
	mv, ok := v.values[name].(map[string]interface{})
	if ok {
		return &mockValue{
			values: mv,
		}
	}
	panic("unsupported value stored under key: " + name)
}
func (v *mockValue) RecordIDs(includeNulls bool, cb func(name string, value istructs.RecordID)) {}
func (v *mockValue) FieldNames(cb func(fieldName string)) {
	for i := range v.values {
		cb(i)
	}
}

type intent struct {
	key   istructs.IStateKeyBuilder
	value istructs.IStateValueBuilder
}

func (s *mockIo) NewValue(key istructs.IStateKeyBuilder) (builder istructs.IStateValueBuilder, err error) {
	k := key.(*mockKeyBuilder)
	if k.storage == storageIoError {
		return nil, errTestIOError
	}
	vb := mockValueBuilder{
		items: make(map[string]interface{}),
	}
	s.intents = append(s.intents, intent{
		key:   key,
		value: &vb,
	})
	return &vb, nil
}

func (s *mockIo) UpdateValue(key istructs.IStateKeyBuilder, existingValue istructs.IStateValue) (builder istructs.IStateValueBuilder, err error) {
	k := key.(*mockKeyBuilder)
	if k.storage == storageIoError {
		return nil, errTestIOError
	}
	vb := mockValueBuilder{
		items: make(map[string]interface{}),
	}
	mv := existingValue.(*mockValue)
	for k, v := range mv.values {
		vb.items[k] = v
	}
	s.intents = append(s.intents, intent{
		key:   key,
		value: &vb,
	})
	return &vb, nil
}

type mockValueBuilder struct {
	items map[string]interface{}
}

func (kb *mockValueBuilder) BuildValue() istructs.IStateValue                 { return nil }
func (kb *mockValueBuilder) PutRecord(name string, record istructs.IRecord)   {}
func (kb *mockValueBuilder) PutEvent(name string, event istructs.IDbEvent)    {}
func (kb *mockValueBuilder) Build() istructs.IValue                           { return nil }
func (kb *mockValueBuilder) PutInt32(name string, value int32)                { kb.items[name] = value }
func (kb *mockValueBuilder) PutInt64(name string, value int64)                {}
func (kb *mockValueBuilder) PutFloat32(name string, value float32)            {}
func (kb *mockValueBuilder) PutFloat64(name string, value float64)            {}
func (kb *mockValueBuilder) PutBytes(name string, value []byte)               { kb.items[name] = value }
func (kb *mockValueBuilder) PutString(name, value string)                     { kb.items[name] = value }
func (kb *mockValueBuilder) PutQName(name string, value istructs.QName)       {}
func (kb *mockValueBuilder) PutBool(name string, value bool)                  {}
func (kb *mockValueBuilder) PutRecordID(name string, value istructs.RecordID) {}

// Tries to make conversion from value to a name type
func (kb *mockValueBuilder) PutNumber(name string, value float64) {}

// Tries to make conversion from value to a name type
func (kb *mockValueBuilder) PutChars(name string, value string) {}
