/*
  - Copyright (c) 2023-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/

package iextenginewazero

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state/stateprovide"
	"github.com/voedger/voedger/pkg/sys"
)

var errTestIOError = errors.New("test i/o error")

var storageTest = appdef.NewQName("sys", "Test")
var storageTest2 = appdef.NewQName("sys", "Test2")
var storageTest3 = appdef.NewQName("sys", "Test3")
var storageTestQname = appdef.NewQName("sys", "TestQName")
var storageIoError = appdef.NewQName("sys", "IoErrorStorage")
var projectorMode bool

const testPackageLocalPath = "testpkg1"
const testPackageFullPath = "github.com/voedger/testpkg1"

type mockIo struct {
	istructs.IState
	istructs.IIntents
	istructs.IPkgNameResolver
	intents []intent
}

func (s *mockIo) PackageFullPath(localName string) string {
	if localName == testPackageLocalPath {
		return testPackageFullPath
	}
	return localName

}
func (s *mockIo) PackageLocalName(fullPath string) string {
	if fullPath == testPackageFullPath {
		return testPackageLocalPath
	}
	return fullPath
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

func (s *mockIo) KeyBuilder(storage, entity appdef.QName) (builder istructs.IStateKeyBuilder, err error) {
	return &mockKeyBuilder{
		entity:  entity,
		storage: storage,
		data:    map[string]interface{}{},
	}, nil
}

func mockedValue(name string, value interface{}) istructs.IStateValue {
	mv := mockValue{
		TestObject: coreutils.TestObject{Data: map[string]interface{}{}},
	}
	mv.Data[name] = value
	return &mv
}

func (s *mockIo) CanExist(key istructs.IStateKeyBuilder) (value istructs.IStateValue, ok bool, err error) {
	k := key.(*mockKeyBuilder)
	mv := mockValue{
		TestObject: coreutils.TestObject{Data: map[string]interface{}{}},
	}
	if k.storage == storageIoError {
		return nil, false, errTestIOError
	}
	if k.storage == sys.Storage_Event {
		if projectorMode {
			mv.Data["offs"] = int32(12345)
			mv.Data["qname"] = "air.UpdateSubscription"
			mv.Data["arg"] = newJSONValue(`
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
		mv.Data["qname"] = "sys.InvitationAccepted"
		mv.Data["arg"] = mockedValue("UserEmail", "email@user.com")
		mv.Data["offs"] = int32(12345)
		return &mv, true, nil
	}
	if k.storage == storageTest {
		mv.Data["500c"] = "01234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789"
		mv.Data["bytes"] = make([]byte, WasmPreallocatedBufferSize*2)
		return &mv, true, nil
	}
	if k.storage == storageTest3 {
		mv.index = make([]interface{}, 4)
		mv.index[0] = int32(123)
		mv.index[1] = "test string"
		mv.index[2] = make([]byte, 1024)
		mv.index[3] = appdef.NewQName(testPackageLocalPath, "test")
		return &mv, true, nil
	}
	if k.storage == storageTest2 {
		const vvv = "012345678901234567890"
		mv.Data["a10"] = vvv
		mv.Data["a11"] = vvv
		mv.Data["a12"] = vvv
		mv.Data["a13"] = vvv
		mv.Data["a14"] = vvv
		mv.Data["a15"] = vvv
		mv.Data["a16"] = vvv
		mv.Data["a17"] = vvv
		mv.Data["a18"] = vvv
		mv.Data["a19"] = vvv
		mv.Data["a20"] = vvv
		mv.Data["a21"] = vvv
		mv.Data["a22"] = vvv
		mv.Data["a23"] = vvv
		mv.Data["a24"] = vvv
		mv.Data["a25"] = vvv
		mv.Data["a26"] = vvv
		mv.Data["a27"] = vvv
		mv.Data["a28"] = vvv
		mv.Data["a29"] = vvv
		mv.Data["a30"] = vvv
		mv.Data["a31"] = vvv
		mv.Data["a32"] = vvv
		mv.Data["a33"] = vvv
		mv.Data["a34"] = vvv
		mv.Data["a35"] = vvv
		mv.Data["a36"] = vvv
		mv.Data["a37"] = vvv
		mv.Data["a38"] = vvv
		mv.Data["a39"] = vvv
		return &mv, true, nil
	}
	if k.storage == storageTestQname {
		qn := k.data["qname"].(appdef.QName)
		if qn.Pkg() != testPackageLocalPath {
			return nil, false, errors.New("unexpected package: " + qn.Pkg())
		}
		return &mv, true, nil
	}
	if k.storage == sys.Storage_SendMail {
		return &mv, true, nil
	}
	if k.storage == sys.Storage_Record {
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
	if k.storage == sys.Storage_Record {
		return nil, stateprovide.ErrNotExists
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
			mk := coreutils.TestObject{Data: map[string]interface{}{}}
			mk.Data["i32"] = int32(i)
			mk.Data["i64"] = 10 + int64(i)
			mk.Data["f32"] = float32(i) + 0.1
			mk.Data["f64"] = float64(i) + 0.01
			mk.Data["str"] = fmt.Sprintf("key%d", i)
			mk.Data["bytes"] = []byte{byte(i), 2, 3}
			mk.Data["qname"] = appdef.NewQName(testPackageLocalPath, fmt.Sprintf("e%d", i))
			mk.Data["bool"] = true

			mv := mockValue{
				TestObject: coreutils.TestObject{Data: map[string]interface{}{}},
			}
			mv.Data["i32"] = 100 + int32(i)
			mv.Data["i64"] = 1000 + int64(i)
			mv.Data["f32"] = float32(i) + 0.001
			mv.Data["f64"] = float64(i) + 0.0001
			mv.Data["str"] = fmt.Sprintf("value%d", i)
			mv.Data["bytes"] = []byte{3, 2, 1}
			mv.Data["qname"] = appdef.NewQName(testPackageLocalPath, fmt.Sprintf("ee%d", i))
			mv.Data["bool"] = false
			if err := callback(&mk, &mv); err != nil {
				return err
			}
		}

	}
	return nil
}

type mockKeyBuilder struct {
	entity  appdef.QName
	storage appdef.QName
	data    map[string]interface{}
}

var _ istructs.IStateKeyBuilder = (*mockKeyBuilder)(nil)

func (kb *mockKeyBuilder) String() string                                   { return "" }
func (kb *mockKeyBuilder) Storage() appdef.QName                            { return kb.storage }
func (kb *mockKeyBuilder) Entity() appdef.QName                             { return kb.entity }
func (kb *mockKeyBuilder) PartitionKey() istructs.IRowWriter                { return nil }
func (kb *mockKeyBuilder) ClusteringColumns() istructs.IRowWriter           { return nil }
func (kb *mockKeyBuilder) Equals(src istructs.IKeyBuilder) bool             { return false }
func (kb *mockKeyBuilder) PutInt8(name string, value int8)                  { kb.data[name] = value }
func (kb *mockKeyBuilder) PutInt16(name string, value int16)                { kb.data[name] = value }
func (kb *mockKeyBuilder) PutInt32(name string, value int32)                {}
func (kb *mockKeyBuilder) PutInt64(name string, value int64)                {}
func (kb *mockKeyBuilder) PutFloat32(name string, value float32)            {}
func (kb *mockKeyBuilder) PutFloat64(name string, value float64)            {}
func (kb *mockKeyBuilder) PutBytes(name string, value []byte)               {}
func (kb *mockKeyBuilder) PutString(name, value string)                     {}
func (kb *mockKeyBuilder) PutQName(name string, value appdef.QName)         { kb.data[name] = value }
func (kb *mockKeyBuilder) PutBool(name string, value bool)                  {}
func (kb *mockKeyBuilder) PutRecordID(name string, value istructs.RecordID) {}
func (kb *mockKeyBuilder) ToBytes(istructs.WSID) ([]byte, []byte, error)    { return nil, nil, nil }

// Tries to make conversion from value to a name type
func (kb *mockKeyBuilder) PutNumber(name string, value json.Number) {}

// Tries to make conversion from value to a name type
func (kb *mockKeyBuilder) PutChars(name string, value string) {}

func (kb *mockKeyBuilder) PutFromJSON(map[string]any) {}

func newJSONValue(jsonString string) istructs.IStateValue {
	v := mockValue{TestObject: coreutils.TestObject{Data: map[string]interface{}{}}}
	err := json.Unmarshal([]byte(jsonString), &v.Data)
	if err != nil {
		panic(err)
	}
	return &v
}

type mockValue struct {
	coreutils.TestObject
	index []interface{}
}

func (v *mockValue) ToJSON(opts ...interface{}) (string, error)     { return "", nil }
func (v *mockValue) AsRecord(name string) (record istructs.IRecord) { return nil }
func (v *mockValue) AsEvent(name string) (event istructs.IDbEvent)  { return nil }

func (v *mockValue) GetAsInt32(index int) int32        { return v.index[index].(int32) }
func (v *mockValue) GetAsInt64(index int) int64        { return 0 }
func (v *mockValue) GetAsFloat32(index int) float32    { return 0 }
func (v *mockValue) GetAsFloat64(index int) float64    { return 0 }
func (v *mockValue) GetAsBytes(index int) []byte       { return v.index[index].([]byte) }
func (v *mockValue) GetAsString(index int) string      { return v.index[index].(string) }
func (v *mockValue) GetAsQName(index int) appdef.QName { return v.index[index].(appdef.QName) }
func (v *mockValue) GetAsBool(index int) bool          { return false }

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
	iv, ok := v.Data[name].(istructs.IStateValue)
	if ok {
		return iv
	}
	mv, ok := v.Data[name].(map[string]interface{})
	if ok {
		return &mockValue{
			TestObject: coreutils.TestObject{Data: mv},
		}
	}
	panic("unsupported value stored under key: " + name)
}
func (v *mockValue) RecordIDs(bool) func(func(string, istructs.RecordID) bool) {
	return func(func(name string, value istructs.RecordID) bool) {}
}
func (v *mockValue) Fields(cb func(iField appdef.IField) bool) { v.TestObject.Fields(cb) }

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
	for k, v := range mv.Data {
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

var _ istructs.IStateValueBuilder = (*mockValueBuilder)(nil)

func (vb *mockValueBuilder) Equal(src istructs.IStateValueBuilder) bool       { return false }
func (vb *mockValueBuilder) BuildValue() istructs.IStateValue                 { return nil }
func (vb *mockValueBuilder) PutRecord(name string, record istructs.IRecord)   {}
func (vb *mockValueBuilder) PutEvent(name string, event istructs.IDbEvent)    {}
func (vb *mockValueBuilder) Build() istructs.IValue                           { return nil }
func (vb *mockValueBuilder) PutInt8(name string, value int8)                  { vb.items[name] = value }
func (vb *mockValueBuilder) PutInt16(name string, value int16)                { vb.items[name] = value }
func (vb *mockValueBuilder) PutInt32(name string, value int32)                { vb.items[name] = value }
func (vb *mockValueBuilder) PutInt64(name string, value int64)                {}
func (vb *mockValueBuilder) PutFloat32(name string, value float32)            {}
func (vb *mockValueBuilder) PutFloat64(name string, value float64)            {}
func (vb *mockValueBuilder) PutBytes(name string, value []byte)               { vb.items[name] = value }
func (vb *mockValueBuilder) PutString(name, value string)                     { vb.items[name] = value }
func (vb *mockValueBuilder) PutQName(name string, value appdef.QName)         { vb.items[name] = value }
func (vb *mockValueBuilder) PutBool(name string, value bool)                  {}
func (vb *mockValueBuilder) PutRecordID(name string, value istructs.RecordID) {}
func (vb *mockValueBuilder) PutFromJSON(map[string]any)                       {}
func (vb *mockValueBuilder) ToBytes() ([]byte, error)                         { return nil, nil }
func (vb *mockValueBuilder) PutNumber(name string, value json.Number)         {}
func (vb *mockValueBuilder) PutChars(name string, value string)               {}
