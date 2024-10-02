/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package coreutils

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMapObject(t *testing.T) {
	require := require.New(t)

	t.Run("basic", func(t *testing.T) {
		mo := MapObject{
			"int641":    json.Number("1"),
			"int642":    json.Number("2"),
			"float64":   json.Number("4"),
			"boolTrue":  true,
			"boolFalse": false,
			"string":    "str",
			"obj": map[string]interface{}{
				"int64": json.Number("5"),
			},
			"objs": []interface{}{
				map[string]interface{}{
					"int64": json.Number("6"),
				},
				map[string]interface{}{
					"int64": json.Number("7"),
				},
			},
		}
		cases := []struct {
			expectedVal interface{}
			f           func() (interface{}, bool, error)
		}{
			{int64(1), func() (interface{}, bool, error) { return mo.AsInt64("int641") }},
			{int64(2), func() (interface{}, bool, error) { return mo.AsInt64("int642") }},
			{float64(4), func() (interface{}, bool, error) { return mo.AsFloat64("float64") }},
			{true, func() (interface{}, bool, error) { return mo.AsBoolean("boolTrue") }},
			{false, func() (interface{}, bool, error) { return mo.AsBoolean("boolFalse") }},
			{"str", func() (interface{}, bool, error) { return mo.AsString("string") }},
			{"str", func() (interface{}, bool, error) {
				val, err := mo.AsStringRequired("string")
				return val, true, err
			}},
			{MapObject{
				"int64": json.Number("5"),
			}, func() (interface{}, bool, error) { return mo.AsObject("obj") }},
			{[]interface{}{
				map[string]interface{}{
					"int64": json.Number("6"),
				},
				map[string]interface{}{
					"int64": json.Number("7"),
				},
			}, func() (interface{}, bool, error) { return mo.AsObjects("objs") }},
		}

		for _, c := range cases {
			val, ok, err := c.f()
			require.NoError(err)
			require.True(ok)
			require.Equal(c.expectedVal, val)
		}
	})

	t.Run("empty", func(t *testing.T) {
		mo := MapObject{
			"fld": nil,
		}
		cases := []struct {
			expected interface{}
			f        func(string) (interface{}, bool, error)
		}{
			{false, func(name string) (interface{}, bool, error) { return mo.AsBoolean(name) }},
			{int64(0), func(name string) (interface{}, bool, error) { return mo.AsInt64(name) }},
			{float64(0), func(name string) (interface{}, bool, error) { return mo.AsFloat64(name) }},
			{MapObject(nil), func(name string) (interface{}, bool, error) { return mo.AsObject(name) }},
			{[]interface{}(nil), func(name string) (interface{}, bool, error) { return mo.AsObjects(name) }},
			{"", func(name string) (interface{}, bool, error) { return mo.AsString(name) }},
		}
		for _, c := range cases {
			val, ok, err := c.f("unknown")
			require.NoError(err)
			require.False(ok)
			require.Equal(c.expected, val)
			val, ok, err = c.f("fld")
			require.NoError(err)
			require.False(ok)
			require.Equal(c.expected, val)
		}
	})

	t.Run("wrong types", func(t *testing.T) {
		mo := MapObject{
			"fld": reflect.TypeOf(0),
		}

		cases := []struct {
			expected interface{}
			f        func() (interface{}, bool, error)
		}{
			{false, func() (interface{}, bool, error) { return mo.AsBoolean("fld") }},
			{float64(0), func() (interface{}, bool, error) { return mo.AsFloat64("fld") }},
			{int64(0), func() (interface{}, bool, error) { return mo.AsInt64("fld") }},
			{MapObject(nil), func() (interface{}, bool, error) { return mo.AsObject("fld") }},
			{[]interface{}(nil), func() (interface{}, bool, error) { return mo.AsObjects("fld") }},
			{"", func() (interface{}, bool, error) { return mo.AsString("fld") }},
			{"", func() (interface{}, bool, error) {
				val, err := mo.AsStringRequired("fld")
				return val, true, err
			}},
		}
		for _, c := range cases {
			val, ok, err := c.f()
			require.ErrorIs(err, ErrFieldTypeMismatch)
			require.True(ok)
			require.Equal(c.expected, val)
		}
	})

	t.Run("AsStringRequired", func(t *testing.T) {
		mo := MapObject{
			"str": nil,
		}
		val, err := mo.AsStringRequired("str")
		require.Empty(val)
		require.ErrorIs(err, ErrFieldsMissed)
		val, err = mo.AsStringRequired("unknown")
		require.Empty(val)
		require.ErrorIs(err, ErrFieldsMissed)
	})

}
