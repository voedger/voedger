/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package coreutils

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

var (
	testQName       = appdef.NewQName("test", "QName")
	testQNameSimple = appdef.NewQName("test", "QNameSimple")
	testFieldDefs   = map[string]appdef.DataKind{
		"int32":    appdef.DataKind_int32,
		"int64":    appdef.DataKind_int64,
		"float32":  appdef.DataKind_float32,
		"float64":  appdef.DataKind_float64,
		"string":   appdef.DataKind_string,
		"bool":     appdef.DataKind_bool,
		"bytes":    appdef.DataKind_bytes,
		"recordID": appdef.DataKind_RecordID,
	}

	testAppDef = func() appdef.IAppDefBuilder {
		app := appdef.New()

		obj := app.AddObject(testQName)
		addFieldDefs(obj, testFieldDefs)

		simpleObj := app.AddObject(testQNameSimple)
		simpleObj.AddField("int32", appdef.DataKind_int32, false)

		return app
	}

	testData = map[string]interface{}{
		"int32":                  int32(1),
		"int64":                  int64(2),
		"float32":                float32(3),
		"float64":                float64(4),
		"string":                 "str",
		"bool":                   true,
		"bytes":                  []byte{5, 6},
		"recordID":               istructs.RecordID(7),
		appdef.SystemField_QName: testQName,
	}
	testDataSimple = map[string]interface{}{
		appdef.SystemField_QName: testQNameSimple,
		"int32":                  int32(42),
	}
	testBasic = func(expectedQName appdef.QName, m map[string]interface{}, require *require.Assertions) {
		require.Equal(int32(1), m["int32"])
		require.Equal(int64(2), m["int64"])
		require.Equal(float32(3), m["float32"])
		require.Equal(float64(4), m["float64"])
		require.Equal("str", m["string"])
		require.Equal(true, m["bool"])
		require.Equal([]byte{5, 6}, m["bytes"])
		require.Equal(istructs.RecordID(7), m["recordID"])
		actualQName, err := appdef.ParseQName(m[appdef.SystemField_QName].(string))
		require.NoError(err)
		require.Equal(expectedQName, actualQName)
	}
)

func addFieldDefs(fields appdef.IFieldsBuilder, fd map[string]appdef.DataKind) {
	for n, k := range fd {
		if !appdef.IsSysField(n) {
			fields.AddField(n, k, false)
		}
	}
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

	appDef := testAppDef()

	t.Run("ObjectToMap", func(t *testing.T) {
		m := ObjectToMap(obj, appDef)
		testBasic(testQName, m, require)
		containerObjects := m["container"].([]map[string]interface{})
		require.Len(containerObjects, 1)
		containerObj := containerObjects[0]
		require.Equal(int32(42), containerObj["int32"])
		require.Equal(testQNameSimple.String(), containerObj[appdef.SystemField_QName])
	})

	t.Run("FieldsToMap", func(t *testing.T) {
		m := FieldsToMap(obj, appDef)
		testBasic(testQName, m, require)
	})

	t.Run("null QName", func(t *testing.T) {
		obj = &TestObject{
			Name: appdef.NullQName,
			Id:   42,
			Data: map[string]interface{}{},
		}
		m := ObjectToMap(obj, appDef)
		require.Empty(m)
		m = FieldsToMap(obj, appDef)
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
	filter := Filter(func(name string, kind appdef.DataKind) bool {
		if name == "bool" {
			require.Equal(appdef.DataKind_bool, kind)
			count++
			return true
		}
		if name == "string" {
			require.Equal(appdef.DataKind_string, kind)
			count++
			return true
		}
		return false
	})

	appDef := testAppDef()

	t.Run("ObjectToMap", func(t *testing.T) {
		m := ObjectToMap(obj, appDef, filter)
		require.Equal(2, count)
		require.Len(m, 2)
		require.Equal(true, m["bool"])
		require.Equal("str", m["string"])
	})

	t.Run("FieldsToMap", func(t *testing.T) {
		m := FieldsToMap(obj, appDef, filter)
		require.Equal(4, count)
		require.Len(m, 2)
		require.Equal(true, m["bool"])
		require.Equal("str", m["string"])
	})
}

func TestMToMap_NonNilsOnly_Filter(t *testing.T) {
	require := require.New(t)
	testDataPartial := map[string]interface{}{
		"int32":                  int32(1),
		"string":                 "str",
		"float32":                float32(2),
		appdef.SystemField_QName: testQName,
	}
	obj := &TestObject{
		Name: testQName,
		Id:   42,
		Data: testDataPartial,
	}
	expected := map[string]interface{}{
		"int32":                  int32(1),
		"string":                 "str",
		appdef.SystemField_QName: testQName.String(),
	}

	appDef := testAppDef()

	t.Run("ObjectToMap", func(t *testing.T) {
		m := ObjectToMap(obj, appDef, WithNonNilsOnly(), Filter(func(name string, kind appdef.DataKind) bool {
			return name != "float32"
		}))
		require.Equal(expected, m)
	})

	t.Run("FieldsToMap", func(t *testing.T) {
		m := FieldsToMap(obj, appDef, WithNonNilsOnly(), Filter(func(name string, kind appdef.DataKind) bool {
			return name != "float32"
		}))
		require.Equal(expected, m)
	})

	t.Run("ObjectToMap + filter", func(t *testing.T) {
		filter := Filter(func(name string, kind appdef.DataKind) bool {
			return name == "string"
		})
		expected := map[string]interface{}{
			"string": "str",
		}
		m := ObjectToMap(obj, appDef, WithNonNilsOnly(), filter)
		require.Equal(expected, m)
	})
}

func TestReadValue(t *testing.T) {
	require := require.New(t)
	iValueFields := map[string]appdef.DataKind{}
	for n, k := range testFieldDefs {
		iValueFields[n] = k
	}
	iValueFields["record"] = appdef.DataKind_Record

	appDefs := testAppDef()
	viewName := appdef.NewQName("test", "view")
	view := appDefs.AddView(viewName)
	view.Key().Partition().AddField("pk", appdef.DataKind_int64)
	view.Key().ClustCols().AddStringField("cc", appdef.DefaultFieldMaxLength)
	for n, k := range iValueFields {
		view.Value().AddField(n, k, false)
	}

	iValueValues := map[string]interface{}{}
	for k, v := range testData {
		iValueValues[k] = v
	}
	iValueValues[appdef.SystemField_QName] = viewName
	iValueValues["record"] = &TestObject{
		Data: testDataSimple,
	}
	iValue := &TestValue{
		TestObject: &TestObject{
			Name: viewName,
			Id:   42,
			Data: iValueValues,
		},
	}

	t.Run("FieldsToMap", func(t *testing.T) {
		m := FieldsToMap(iValue, appDefs)
		testBasic(viewName, m, require)
		require.Equal(map[string]interface{}{"int32": int32(42), appdef.SystemField_QName: "test.QNameSimple"}, m["record"])
	})

	t.Run("FieldsToMap non-nils only", func(t *testing.T) {
		m := FieldsToMap(iValue, appDefs, WithNonNilsOnly())
		testBasic(viewName, m, require)
		require.Equal(map[string]interface{}{"int32": int32(42), appdef.SystemField_QName: "test.QNameSimple"}, m["record"])
	})

	t.Run("panic if an object contains DataKind_Record field but is not IValue", func(t *testing.T) {
		obj := &TestObject{
			Name: testQName,
			Data: iValueValues,
		}
		require.Panics(func() { FieldsToMap(obj, appDefs) })
		require.Panics(func() { FieldsToMap(obj, appDefs, WithNonNilsOnly()) })
	})
}

func TestObjectReaderErrors(t *testing.T) {
	require := require.New(t)
	require.Panics(func() { ReadByKind("", appdef.DataKind_FakeLast, nil) })
}
