/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package coreutils

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/istructs"
)

var (
	testQName       = istructs.NewQName("test", "QName")
	testQNameSimple = istructs.NewQName("test", "QNameSimple")
	testFields      = map[string]istructs.DataKindType{
		"int32":                    istructs.DataKind_int32,
		"int64":                    istructs.DataKind_int64,
		"float32":                  istructs.DataKind_float32,
		"float64":                  istructs.DataKind_float64,
		"string":                   istructs.DataKind_string,
		"bool":                     istructs.DataKind_bool,
		"bytes":                    istructs.DataKind_bytes,
		"recordID":                 istructs.DataKind_RecordID,
		istructs.SystemField_QName: istructs.DataKind_QName,
	}
	schema = TestSchema{
		Fields_: testFields,
		QName_:  testQName,
	}
	schemaSimple = TestSchema{
		Fields_: map[string]istructs.DataKindType{
			istructs.SystemField_QName: istructs.DataKind_QName,
			"int32":                    istructs.DataKind_int32,
		},
	}
	schemas = TestSchemas{Schemas_: map[istructs.QName]istructs.ISchema{
		testQName:       schema,
		testQNameSimple: schemaSimple,
	}}
	testData = map[string]interface{}{
		"int32":                    int32(1),
		"int64":                    int64(2),
		"float32":                  float32(3),
		"float64":                  float64(4),
		"string":                   "str",
		"bool":                     true,
		"bytes":                    []byte{5, 6},
		"recordID":                 istructs.RecordID(7),
		istructs.SystemField_QName: testQName,
	}
	testDataSimple = map[string]interface{}{
		istructs.SystemField_QName: testQNameSimple,
		"int32":                    int32(42),
	}
	testBasic = func(m map[string]interface{}, require *require.Assertions) {
		require.Equal(int32(1), m["int32"])
		require.Equal(int64(2), m["int64"])
		require.Equal(float32(3), m["float32"])
		require.Equal(float64(4), m["float64"])
		require.Equal("str", m["string"])
		require.Equal(true, m["bool"])
		require.Equal([]byte{5, 6}, m["bytes"])
		require.Equal(istructs.RecordID(7), m["recordID"])
		actualQName, err := istructs.ParseQName(m[istructs.SystemField_QName].(string))
		require.NoError(err)
		require.Equal(testQName, actualQName)
	}
)

func TestNewSchemaFields(t *testing.T) {
	qName := istructs.NewQName("test", "qname")
	testFields := map[string]istructs.DataKindType{
		"fld1": istructs.DataKind_int32,
		"str":  istructs.DataKind_string,
	}
	s := TestSchema{
		Fields_: testFields,
		QName_:  qName,
	}
	sf := NewSchemaFields(s)
	require.Equal(t, SchemaFields(testFields), sf)
}

func TestToMap_Basic(t *testing.T) {
	require := require.New(t)
	obj := &TestObject{
		Name: testQName,
		Id:   42,
		Data: testData,
		Containers_: map[string][]*TestObject{
			"container": {
				{
					Name: testQNameSimple,
					Data: testDataSimple,
				},
			},
		},
	}

	t.Run("ObjectToMap", func(t *testing.T) {
		m := ObjectToMap(obj, schemas)
		testBasic(m, require)
		containerObjs := m["container"].([]map[string]interface{})
		require.Len(containerObjs, 1)
		containerObj := containerObjs[0]
		require.Equal(int32(42), containerObj["int32"])
		require.Equal(testQNameSimple.String(), containerObj[istructs.SystemField_QName])
	})

	t.Run("FieldsToMap", func(t *testing.T) {
		m := FieldsToMap(obj, schemas)
		testBasic(m, require)
	})

	t.Run("Read", func(t *testing.T) {
		sf := NewSchemaFields(schema)
		m := map[string]interface{}{}
		for fieldName := range sf {
			m[fieldName] = Read(fieldName, sf, obj)
		}
		testBasic(m, require)
	})

	t.Run("null QName", func(t *testing.T) {
		obj = &TestObject{
			Name: testQName,
			Id:   42,
			Data: map[string]interface{}{
				istructs.SystemField_QName: istructs.NullQName,
			},
		}
		m := ObjectToMap(obj, schemas)
		require.Empty(m)
		m = FieldsToMap(obj, schemas)
		require.Empty(m)
	})
}

func TestToMap_Filter(t *testing.T) {
	require := require.New(t)
	obj := &TestObject{
		Name: testQName,
		Id:   42,
		Data: testData,
	}

	count := 0
	filter := Filter(func(name string, kind istructs.DataKindType) bool {
		if name == "bool" {
			require.Equal(istructs.DataKind_bool, kind)
			count++
			return true
		}
		if name == "string" {
			require.Equal(istructs.DataKind_string, kind)
			count++
			return true
		}
		return false
	})

	t.Run("ObjectToMap", func(t *testing.T) {
		m := ObjectToMap(obj, schemas, filter)
		require.Equal(2, count)
		require.Len(m, 2)
		require.Equal(true, m["bool"])
		require.Equal("str", m["string"])
	})

	t.Run("FieldsToMap", func(t *testing.T) {
		m := FieldsToMap(obj, schemas, filter)
		require.Equal(4, count)
		require.Len(m, 2)
		require.Equal(true, m["bool"])
		require.Equal("str", m["string"])
	})
}

func TestMToMap_NonNilsOnly_Filter(t *testing.T) {
	require := require.New(t)
	testDataPartial := map[string]interface{}{
		"int32":                    int32(1),
		"string":                   "str",
		"float32":                  float32(2),
		istructs.SystemField_QName: testQName,
	}
	obj := &TestObject{
		Name: testQName,
		Id:   42,
		Data: testDataPartial,
	}
	expected := map[string]interface{}{
		"int32":                    int32(1),
		"string":                   "str",
		istructs.SystemField_QName: testQName.String(),
	}

	t.Run("OjectToMap", func(t *testing.T) {
		m := ObjectToMap(obj, schemas, WithNonNilsOnly(), Filter(func(name string, kind istructs.DataKindType) bool {
			return name != "float32"
		}))
		require.Equal(expected, m)
	})

	t.Run("FieldsToMap", func(t *testing.T) {
		m := FieldsToMap(obj, schemas, WithNonNilsOnly(), Filter(func(name string, kind istructs.DataKindType) bool {
			return name != "float32"
		}))
		require.Equal(expected, m)
	})

	t.Run("OjectToMap + filter", func(t *testing.T) {
		filter := Filter(func(name string, kind istructs.DataKindType) bool {
			return name == "string"
		})
		expected := map[string]interface{}{
			"string": "str",
		}
		m := ObjectToMap(obj, schemas, WithNonNilsOnly(), filter)
		require.Equal(expected, m)
	})
}

func TestReadValue(t *testing.T) {
	require := require.New(t)
	iValueFields := map[string]istructs.DataKindType{}
	for k, v := range testFields {
		iValueFields[k] = v
	}
	iValueFields["record"] = istructs.DataKind_Record
	iValueSchema := TestSchema{
		QName_:  testQName,
		Fields_: iValueFields,
	}
	iValueValues := map[string]interface{}{}
	for k, v := range testData {
		iValueValues[k] = v
	}
	iValueValues["record"] = &TestObject{
		Data: testDataSimple,
	}
	schemasWithIValue := TestSchemas{
		Schemas_: map[istructs.QName]istructs.ISchema{
			testQName:       iValueSchema,
			testQNameSimple: schemaSimple,
		},
	}
	iValue := &TestValue{
		TestObject: &TestObject{
			Name: testQName,
			Id:   42,
			Data: iValueValues,
		},
	}
	t.Run("ReadValue", func(t *testing.T) {
		sf := NewSchemaFields(iValueSchema)
		actual := map[string]interface{}{}
		for fieldName := range iValueValues {
			actual[fieldName] = ReadValue(fieldName, sf, schemasWithIValue, iValue)
		}
		testBasic(actual, require)
		require.Equal(map[string]interface{}{"int32": int32(42), istructs.SystemField_QName: "test.QNameSimple"}, actual["record"])
	})

	t.Run("FieldsToMap", func(t *testing.T) {
		m := FieldsToMap(iValue, schemasWithIValue)
		testBasic(m, require)
		require.Equal(map[string]interface{}{"int32": int32(42), istructs.SystemField_QName: "test.QNameSimple"}, m["record"])
	})

	t.Run("FieldsToMap non-nils only", func(t *testing.T) {
		m := FieldsToMap(iValue, schemasWithIValue, WithNonNilsOnly())
		testBasic(m, require)
		require.Equal(map[string]interface{}{"int32": int32(42), istructs.SystemField_QName: "test.QNameSimple"}, m["record"])
	})

	t.Run("panic if an object contains DataKind_Record field but is not IValue", func(t *testing.T) {
		obj := &TestObject{
			Name: testQName,
			Data: iValueValues,
		}
		require.Panics(func() { FieldsToMap(obj, schemasWithIValue) })
		require.Panics(func() { FieldsToMap(obj, schemasWithIValue, WithNonNilsOnly()) })
	})
}

func TestObjectReaderErrors(t *testing.T) {
	require := require.New(t)
	require.Panics(func() { ReadByKind("", istructs.DataKind_FakeLast, nil) })
}
