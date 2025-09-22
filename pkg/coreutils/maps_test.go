/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package coreutils

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/istructs"
)

func TestBasicUsage_PairsToMap(t *testing.T) {
	require := require.New(t)
	cases := []struct {
		pairs       []string
		expectedMap map[string]string
	}{
		{nil, map[string]string{}},
		{[]string{}, map[string]string{}},
		{[]string{"="}, map[string]string{"": ""}},
		{[]string{"=v"}, map[string]string{"": "v"}},
		{[]string{"k="}, map[string]string{"k": ""}},
		{[]string{"k=v"}, map[string]string{"k": "v"}},
		{[]string{"k1=v1", "k2=v2"}, map[string]string{"k1": "v1", "k2": "v2"}},
		{[]string{"k=v", "k=v"}, map[string]string{"k": "v"}},
	}

	for _, c := range cases {
		m := map[string]string{}
		require.NoError(PairsToMap(c.pairs, m))
		require.Equal(c.expectedMap, m)
	}
}

func TestPairsToMapErrors(t *testing.T) {
	require := require.New(t)
	cases := [][]string{
		{""},
		{"k"},
		{"=="},
		{"k=v="},
	}

	for _, c := range cases {
		m := map[string]string{}
		require.Error(PairsToMap(c, m))
	}
}

func TestMapToObject(t *testing.T) {
	o := &TestObject{
		Data: map[string]interface{}{},
	}
	require := require.New(t)
	require.NoError(MapToObject(map[string]interface{}{
		"float64":  float64(42),
		"float32":  float32(43),
		"int32":    int32(44),
		"int64":    int64(45),
		"recordID": istructs.RecordID(46),
		"str":      "str1",
		"bool":     true,
		"any":      nil, // will be ignored
		"numInt":   json.Number("123"),
		"numFloat": json.Number("123.45"),
	}, o))
	require.Len(o.Data, 9)
	require.Equal(float64(42), o.AsFloat64("float64"))
	require.Equal(float32(43), o.AsFloat32("float32"))
	require.Equal(int32(44), o.AsInt32("int32"))
	require.Equal(int64(45), o.AsInt64("int64"))
	require.Equal(istructs.RecordID(46), o.AsRecordID("recordID"))
	require.Equal("str1", o.AsString("str"))
	v := o.AsBool("bool")
	require.True(v)
	require.Equal(int32(123), o.AsInt32("numInt"))
	require.Equal(float32(123.45), o.AsFloat32("numFloat"))

	require.Error(MapToObject(map[string]interface{}{"fld": 42}, o))
}

func TestMergeMaps(t *testing.T) {
	require := require.New(t)

	t.Run("zero maps", func(t *testing.T) {
		m := MergeMaps()
		require.Empty(m)
	})

	t.Run("one map", func(t *testing.T) {
		m1 := map[string]interface{}{"a": 1}
		m := MergeMaps(m1)
		require.Equal(map[string]interface{}{"a": 1}, m)
	})

	t.Run("multiple maps, overlapping keys", func(t *testing.T) {
		m1 := map[string]interface{}{"a": 1}
		m2 := map[string]interface{}{"b": 2, "a": 3}
		m3 := map[string]interface{}{"c": 4}
		m := MergeMaps(m1, m2, m3)
		require.Equal(map[string]interface{}{"a": 3, "b": 2, "c": 4}, m)
	})

	t.Run("nil maps", func(t *testing.T) {
		m1 := map[string]interface{}{"a": 1}
		m := MergeMaps(nil, m1, nil)
		require.Equal(map[string]interface{}{"a": 1}, m)
	})
}
