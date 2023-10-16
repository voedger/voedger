/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package coreutils

import (
	"testing"

	"github.com/stretchr/testify/require"
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

func TestMarshal(t *testing.T) {
	o := &TestObject{
		Data: map[string]interface{}{},
	}
	require := require.New(t)
	require.NoError(Marshal(o, map[string]interface{}{
		"float64": float64(42),
		"str":     "str1",
		"bool":    true,
		"any":     nil, // will be ignored
	}))
	require.Len(o.Data, 3)
	require.Equal(float64(42), o.Data["float64"])
	require.Equal("str1", o.Data["str"])
	require.Equal(true, o.Data["bool"])

	require.Error(Marshal(o, map[string]interface{}{"fld": 42}))
}
