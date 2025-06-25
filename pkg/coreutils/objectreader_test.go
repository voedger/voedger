/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package coreutils

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/istructs"
)

var (
	testWS          = appdef.NewQName("test", "test_ws")
	testQName       = appdef.NewQName("test", "QName")
	testQNameSimple = appdef.NewQName("test", "QNameSimple")
	testQNameView   = appdef.NewQName("test", "view")
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
		ID_:  42,
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

	appDef := testAppDef(t)

	t.Run("ObjectToMap", func(t *testing.T) {
		m := ObjectToMap(obj, appDef)
		testBasic(testQName, m, require)
		containerObjects := m["container"].([]map[string]interface{})
		require.Len(containerObjects, 1)
		containerObj := containerObjects[0]
		require.Equal(int32(42), containerObj["int32"])
		require.Equal(testQNameSimple, containerObj[appdef.SystemField_QName])
	})

	t.Run("FieldsToMap", func(t *testing.T) {
		m := FieldsToMap(obj, appDef)
		testBasic(testQName, m, require)
	})

	t.Run("null QName", func(t *testing.T) {
		obj := &TestObject{
			Name: appdef.NullQName,
			ID_:  42,
			Data: map[string]interface{}{},
		}
		m := ObjectToMap(obj, appDef)
		require.Empty(m)
		m = FieldsToMap(obj, appDef)
		require.Empty(m)
	})

	t.Run("all fields", func(t *testing.T) {
		m := ObjectToMap(obj, appDef, WithAllFields())
		testBasic(testQName, m, require)
	})
}

func TestToMap_Filter(t *testing.T) {
	require := require.New(t)
	obj := &TestObject{
		Name: testQName,
		ID_:  42,
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

	appDef := testAppDef(t)

	t.Run("ObjectToMap", func(t *testing.T) {
		m := ObjectToMap(obj, appDef, filter)
		require.Equal(2, count)
		require.Len(m, 2)
		v, ok := m["bool"].(bool)
		require.True(ok)
		require.True(v)
		require.Equal("str", m["string"])
	})

	t.Run("FieldsToMap", func(t *testing.T) {
		m := FieldsToMap(obj, appDef, filter)
		require.Equal(4, count)
		require.Len(m, 2)
		v, ok := m["bool"].(bool)
		require.True(ok)
		require.True(v)
		require.Equal("str", m["string"])
	})
}

func TestReadValue(t *testing.T) {
	require := require.New(t)

	appDef := testAppDef(t)

	iValueValues := map[string]interface{}{}
	for k, v := range testData {
		iValueValues[k] = v
	}
	iValueValues[appdef.SystemField_QName] = testQNameView
	iValueValues["record"] = &TestObject{
		Data: testDataSimple,
		Name: testQNameSimple,
	}
	iValue := &TestValue{
		TestObject: &TestObject{
			Name: testQNameView,
			ID_:  42,
			Data: iValueValues,
		},
	}

	t.Run("FieldsToMap", func(t *testing.T) {
		m := FieldsToMap(iValue, appDef)
		testBasic(testQNameView, m, require)
		require.Equal(
			map[string]interface{}{
				"int32":                     int32(42),
				appdef.SystemField_QName:    testQNameSimple,
				appdef.SystemField_ID:       istructs.RecordID(0),
				appdef.SystemField_IsActive: true,
			},
			m["record"],
		)
	})
}

func TestObjectReaderErrors(t *testing.T) {
	require := require.New(t)
	require.Panics(func() { ReadByKind("", appdef.DataKind_FakeLast, nil) })
}

func TestJSONMapToCUDBody(t *testing.T) {
	t.Run("basic usage", func(t *testing.T) {
		data := []map[string]interface{}{
			{
				"fld1": "val1",
			},
			{
				"fld2": "val2",
			},
		}
		cudBody := JSONMapToCUDBody(data)
		require.JSONEq(t, `{"cuds":[{"fields":{"fld1":"val1"}},{"fields":{"fld2":"val2"}}]}`, cudBody)
	})
	t.Run("failed to marshel -> panic", func(t *testing.T) {
		data := []map[string]interface{}{
			{
				"fld1": func() {},
			},
		}
		require.Panics(t, func() { JSONMapToCUDBody(data) })
	})
}

func TestCheckValueByKind(t *testing.T) {
	okCases := []struct {
		val  any
		kind appdef.DataKind
	}{
		{int8(1), appdef.DataKind_int8},
		{int16(2), appdef.DataKind_int16},
		{int32(3), appdef.DataKind_int32},
		{int64(4), appdef.DataKind_int64},
		{int64(5), appdef.DataKind_RecordID},
		{float32(6), appdef.DataKind_float32},
		{float64(7), appdef.DataKind_float64},
		{true, appdef.DataKind_bool},
		{"str", appdef.DataKind_string},
		{"str.str", appdef.DataKind_QName},
		{[]byte{9}, appdef.DataKind_bytes},
		{istructs.RecordID(10), appdef.DataKind_RecordID},
		{int64(11), appdef.DataKind_RecordID},
		{appdef.NewQName("1", "1"), appdef.DataKind_QName},
	}
	for _, c := range okCases {
		t.Run(fmt.Sprintf("%v", c.val), func(t *testing.T) {
			require.NoError(t, CheckValueByKind(c.val, c.kind))
		})
	}

	t.Run("not ok", func(t *testing.T) {
		for kind := appdef.DataKind(1); kind < appdef.DataKind_FakeLast; kind++ {
			t.Run(kind.String(), func(t *testing.T) {
				require.Error(t, CheckValueByKind(func() {}, kind))
			})
		}
	})
}

func testBasic(expectedQName appdef.QName, m map[string]interface{}, require *require.Assertions) {
	require.Equal(int32(1), m["int32"])
	require.Equal(int64(2), m["int64"])
	require.Equal(float32(3), m["float32"])
	require.Equal(float64(4), m["float64"])
	require.Equal("str", m["string"])
	v, ok := m["bool"].(bool)
	require.True(ok)
	require.True(v)
	require.Equal([]byte{5, 6}, m["bytes"])
	require.Equal(istructs.RecordID(7), m["recordID"])
	actualQName := m[appdef.SystemField_QName].(appdef.QName)
	require.Equal(expectedQName, actualQName)
}

func testAppDef(t *testing.T) appdef.IAppDef {
	adb := builder.New()

	wsb := adb.AddWorkspace(testWS)

	obj := wsb.AddObject(testQName)
	addFieldDefs(obj, testFieldDefs)

	simpleObj := wsb.AddObject(testQNameSimple)
	simpleObj.AddField("int32", appdef.DataKind_int32, false)

	view := wsb.AddView(testQNameView)
	view.Key().PartKey().AddField("pk", appdef.DataKind_int64)
	view.Key().ClustCols().AddField("cc", appdef.DataKind_string)
	iValueFields := map[string]appdef.DataKind{}
	for n, k := range testFieldDefs {
		iValueFields[n] = k
	}
	iValueFields["record"] = appdef.DataKind_Record
	for n, k := range iValueFields {
		view.Value().AddField(n, k, false)
	}

	app, err := adb.Build()
	require.NoError(t, err)

	return app
}

func TestCUDsToMap(t *testing.T) {
	require := require.New(t)
	appDef := testAppDef(t)

	t.Run("single CUD", func(t *testing.T) {
		rec := &TestObject{
			Name:   testQName,
			ID_:    1,
			IsNew_: true,
			Data: map[string]interface{}{
				"int32":                  int32(42),
				appdef.SystemField_QName: testQName,
			},
		}
		event := &mockDbEvent{cuds: []istructs.ICUDRow{rec}}
		got := CUDsToMap(event, appDef)
		require.Len(got, 1)
		cud := got[0]
		require.Equal(istructs.RecordID(1), cud["sys.ID"])
		require.Equal(testQName.String(), cud["sys.QName"])
		require.Equal(true, cud["IsNew"])
		fields := cud["fields"].(map[string]interface{})
		require.Equal(int32(42), fields["int32"])
	})

	t.Run("multiple CUDs", func(t *testing.T) {
		rec1 := &TestObject{Name: testQName, ID_: 1, Data: map[string]interface{}{"int32": int32(1)}}
		rec2 := &TestObject{Name: testQName, ID_: 2, Data: map[string]interface{}{"int32": int32(2)}}
		event := &mockDbEvent{cuds: []istructs.ICUDRow{rec1, rec2}}
		got := CUDsToMap(event, appDef)
		require.Len(got, 2)
	})

	t.Run("filter", func(t *testing.T) {
		rec1 := &TestObject{Name: testQName, ID_: 1, Data: map[string]interface{}{"int32": int32(1)}}
		rec2 := &TestObject{Name: testQNameSimple, ID_: 2, Data: map[string]interface{}{"int32": int32(2)}}
		event := &mockDbEvent{cuds: []istructs.ICUDRow{rec1, rec2}}
		got := CUDsToMap(event, appDef, WithFilter(func(qn appdef.QName) bool { return qn == testQName }))
		require.Len(got, 1)
		require.Equal(testQName.String(), got[0]["sys.QName"])
	})

	t.Run("WithMapperOpts", func(t *testing.T) {
		rec := &TestObject{Name: testQName, ID_: 1, Data: map[string]interface{}{
			"int32":                  int32(42),
			"string":                 "foo",
			appdef.SystemField_QName: testQName}}
		event := &mockDbEvent{cuds: []istructs.ICUDRow{rec}}
		got := CUDsToMap(event, appDef, WithMapperOpts(Filter(func(name string, kind appdef.DataKind) bool { return name == "string" })))
		fields := got[0]["fields"].(map[string]interface{})
		require.Equal("foo", fields["string"])
		require.NotContains(fields, "int32")
	})
}

type mockDbEvent struct {
	istructs.IDbEvent
	cuds []istructs.ICUDRow
}

func (e *mockDbEvent) CUDs(cb func(istructs.ICUDRow) bool) {
	for _, c := range e.cuds {
		if !cb(c) {
			break
		}
	}
}
