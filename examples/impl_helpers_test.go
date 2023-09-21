/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */

package examples

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

const (
	testTransactionID = "1"
	testTableNo       = "25"
)

var mockTableRest int64
var storageEvent = appdef.NewQName("sys", "EventStorage")
var transactionQName = appdef.NewQName("vrestaurant", "Transaction")
var tableStatusQName = appdef.NewQName("vrestaurant", "TableStatus")
var orderQName = appdef.NewQName("vrestaurant", "Order")
var billQName = appdef.NewQName("vrestaurant", "Bill")

var mockmode int
var mv mockValue

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

func (s *mockIo) KeyBuilder(storage, entity appdef.QName) (builder istructs.IStateKeyBuilder, err error) {
	return &mockKeyBuilder{
		entity:  entity,
		storage: storage,
	}, nil
}

func mockedValue() *mockValue {
	mv := mockValue{
		TestObject: coreutils.TestObject{Data: map[string]interface{}{}},
	}
	return &mv
}

func (s *mockIo) CanExist(key istructs.IStateKeyBuilder) (value istructs.IStateValue, ok bool, err error) {
	k := key.(*mockKeyBuilder)
	if k.storage == tableStatusQName {
		mv.Data["NotPaid"] = mockTableRest
		if mockTableRest > 0 {
			mv.Data["Status"] = int32(1)
		} else {
			mv.Data["Status"] = int32(0)
		}
		return &mv, true, nil
	}
	if k.storage == transactionQName {
		mv.Data["qname"] = transactionQName
		mv.Data["arg"] = newJsonValue(`
			{
				"transactionID": 1,
				"Tableno": 25
			}
			`)
		return &mv, true, nil
	}
	if k.storage == storageEvent {
		if mockmode == modeOrder {
			mockTableRest = 1560
			mv.Data["qname"] = orderQName
			mv.Data["arg"] = newJsonValue(`
				{
					"transactionID": 1, 
					"OrderItem": [
						{
						  "Quantity": 2,
						  "Price": 250
						},
						{
   						  "Quantity": 1,
				  		  "Price": 130
					    },
						{
  						  "Quantity": 3,
				  		  "Price": 310
					    }
					]
				}
			`)
		} else if mockmode == modeBill {
			mockTableRest = 0
			mv.Data["qname"] = billQName
			mv.Data["arg"] = newJsonValue(`
				{
					"transactionID": 1, 
					"BillPayment": [
						{
						  "Kind": 1,
						  "Amount": 700
						},
						{
  						  "Kind": 2,
				  		  "Amount": 860
					    }
					]
				}
			`)
		} else if mockmode == modeBill1 {
			mockTableRest = 860
			mv.Data["qname"] = billQName
			mv.Data["arg"] = newJsonValue(`
					{
						"transactionID": 1, 
						"BillPayment": [
							{
							  "Kind": 1,
							  "Amount": 700
							}
						]
					}
				`)
		} else if mockmode == modeBill2 {
			mockTableRest = 0
			mv.Data["qname"] = billQName
			mv.Data["arg"] = newJsonValue(`
						{
							"transactionID": 1, 
							"BillPayment": [
								{
									"Kind": 2,
									"Amount": 860
								}
							]
						}
					`)
		}
		return &mv, true, nil
	}
	return nil, false, errors.New("unsupported storage: " + k.storage.Pkg() + "." + k.storage.Entity())
}

func (s *mockIo) CanExistAll(keys []istructs.IStateKeyBuilder, callback istructs.StateValueCallback) (err error) {
	return nil
}

func (s *mockIo) MustExist(key istructs.IStateKeyBuilder) (value istructs.IStateValue, err error) {
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

type mockKeyBuilder struct {
	entity  appdef.QName
	storage appdef.QName
}

func (kb *mockKeyBuilder) Storage() appdef.QName                            { return kb.storage }
func (kb *mockKeyBuilder) Entity() appdef.QName                             { return kb.entity }
func (kb *mockKeyBuilder) PartitionKey() istructs.IRowWriter                { return nil }
func (kb *mockKeyBuilder) ClusteringColumns() istructs.IRowWriter           { return nil }
func (kb *mockKeyBuilder) Equals(src istructs.IKeyBuilder) bool             { return false }
func (kb *mockKeyBuilder) PutInt32(name string, value int32)                {}
func (kb *mockKeyBuilder) PutInt64(name string, value int64)                {}
func (kb *mockKeyBuilder) PutFloat32(name string, value float32)            {}
func (kb *mockKeyBuilder) PutFloat64(name string, value float64)            {}
func (kb *mockKeyBuilder) PutBytes(name string, value []byte)               {}
func (kb *mockKeyBuilder) PutString(name, value string)                     {}
func (kb *mockKeyBuilder) PutQName(name string, value appdef.QName)         {}
func (kb *mockKeyBuilder) PutBool(name string, value bool)                  {}
func (kb *mockKeyBuilder) PutRecordID(name string, value istructs.RecordID) {}

// Tries to make conversion from value to a name type
func (kb *mockKeyBuilder) PutNumber(name string, value float64) {}

// Tries to make conversion from value to a name type
func (kb *mockKeyBuilder) PutChars(name string, value string) {}

func newJsonValue(jsonString string) istructs.IStateValue {
	v := mockValue{TestObject: coreutils.TestObject{Data: map[string]interface{}{}}}

	err := json.Unmarshal([]byte(jsonString), &v.Data)
	if err != nil {
		panic(err)
	}
	convertMapFlot64ToInt64(v.Data)
	return &v
}

func convertMapFlot64ToInt64(val map[string]interface{}) {
	for key, value := range val {
		if fval, ok := value.(float64); ok {
			val[key] = int64(fval)
		} else if fval, ok := value.([]interface{}); ok {
			for _, intf := range fval {
				convertArrFlot64ToInt64(intf)
			}
		}
	}
}

func convertArrFlot64ToInt64(val interface{}) {
	if v, ok := val.(float64); ok {
		val = int64(v)
	} else if fval, ok := val.([]interface{}); ok {
		for _, intf := range fval {
			convertArrFlot64ToInt64(intf)
		}
	} else if mval, ok := val.(map[string]interface{}); ok {
		convertMapFlot64ToInt64(mval)
	}
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
func (v *mockValue) GetAsQName(index int) appdef.QName { return appdef.NullQName }
func (v *mockValue) GetAsBool(index int) bool          { return false }

func (v *mockValue) Length() int {
	if v.index == nil {
		return 0
	}
	return len(v.index)
}

func (v *mockValue) AsRecordID(name string) istructs.RecordID { return 0 }
func (v *mockValue) GetAsValue(index int) istructs.IStateValue {
	iv, ok := v.index[index].(istructs.IStateValue)
	if ok {
		return iv
	}
	mv, ok := v.index[index].(map[string]interface{})
	if ok {
		return &mockValue{
			TestObject: coreutils.TestObject{Data: mv},
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
	} else {
		mv, ok := v.Data[name].([]interface{})
		if ok {
			return &mockValue{
				TestObject: coreutils.TestObject{Data: nil},
				index:      mv,
			}
		}
	}
	panic("unsupported value stored under key: " + name)
}
func (v *mockValue) RecordIDs(includeNulls bool, cb func(name string, value istructs.RecordID)) {}
func (v *mockValue) FieldNames(cb func(fieldName string)) {
	v.TestObject.FieldNames(cb)
}

type intent struct {
	key   istructs.IStateKeyBuilder
	value istructs.IStateValueBuilder
}

func (s *mockIo) NewValue(key istructs.IStateKeyBuilder) (builder istructs.IStateValueBuilder, err error) {
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
func (kb *mockValueBuilder) PutQName(name string, value appdef.QName)         {}
func (kb *mockValueBuilder) PutBool(name string, value bool)                  {}
func (kb *mockValueBuilder) PutRecordID(name string, value istructs.RecordID) {}

// Tries to make conversion from value to a name type
func (kb *mockValueBuilder) PutNumber(name string, value float64) {}

// Tries to make conversion from value to a name type
func (kb *mockValueBuilder) PutChars(name string, value string) {}

// InitTestBO adds departments, articles, payment types in storage
func InitTestBO() {
	// TODO: fill storage with BO data
}

// CreateTestOrder fill mock Order structure
func CreateTestOrder() {
	// TODO: fill order structure
}

// CreateTestBill fill mock Bill structure
func CreateTestBill() {
	// TODO: fill bill structure
}
