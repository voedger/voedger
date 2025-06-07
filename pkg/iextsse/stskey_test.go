/*
 * Copyright (c) 2025-present unTill Software Development Group B. V. 
 * @author Maxim Geraskin
 */


package iextsse

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_STSKey_NamespaceAndName(t *testing.T) {
	key := NewSTSKey("myNamespace", "myKey")
	require.Equal(t, "myNamespace", key.Namespace(), "Namespace should be 'myNamespace'")
	require.Equal(t, "myKey", key.Name(), "Name should be 'myKey'")
}

func Test_STSKey_SetAndGetInt64(t *testing.T) {
	key := NewSTSKey("namespace", "name")
	key.SetInt64("age1", 25)
	key.SetInt64("age2", 35)

	{
		value := key.AsInt64("age1")
		require.Equal(t, int64(25), value, "Value for 'age' should be 25")
	}
	{
		value := key.AsInt64("age2")
		require.Equal(t, int64(35), value, "Value for 'age' should be 35")
	}
}

func Test_STSKey_SetAndGetString(t *testing.T) {
	key := NewSTSKey("namespace", "name")
	key.SetString("username1", "alice")
	key.SetString("username2", "bob")

	{
		value := key.AsString("username1")
		require.Equal(t, "alice", value, "Value for 'username1' should be 'alice'")
	}
	{
		value := key.AsString("username2")
		require.Equal(t, "bob", value, "Value for 'username2' should be 'bob'")
	}
}

func Test_STSKey_AsInt64MissingKey(t *testing.T) {
	key := NewSTSKey("namespace", "name")

	require.PanicsWithValue(t, "value missing for key 'nonexistent'", func() {
		key.AsInt64("nonexistent")
	}, "AsInt64 should panic when key is missing")
}

func Test_STSKey_AsStringMissingKey(t *testing.T) {
	key := NewSTSKey("namespace", "name")

	require.PanicsWithValue(t, "value missing for key 'nonexistent'", func() {
		key.AsString("nonexistent")
	}, "AsString should panic when key is missing")
}

func Test_STSKey_AsInt64WrongType(t *testing.T) {
	key := NewSTSKey("namespace", "name")
	key.SetString("age", "twenty-five")

	require.PanicsWithValue(t, "value for key 'age' is not int64", func() {
		key.AsInt64("age")
	}, "AsInt64 should panic when value is not int64")
}

func Test_STSKey_AsStringWrongType(t *testing.T) {
	key := NewSTSKey("namespace", "name")
	key.SetInt64("active", 1)

	require.PanicsWithValue(t, "value for key 'active' is not string", func() {
		key.AsString("active")
	}, "AsString should panic when value is not string")
}

func Test_STSKey_MultipleValues(t *testing.T) {
	key := NewSTSKey("namespace", "name")
	key.SetInt64("count", 10)
	key.SetString("status", "active")

	require.Equal(t, int64(10), key.AsInt64("count"), "Value for 'count' should be 10")
	require.Equal(t, "active", key.AsString("status"), "Value for 'status' should be 'active'")
}

func Test_STSKey_OverwriteValue(t *testing.T) {
	key := NewSTSKey("namespace", "name")
	key.SetInt64("value", 100)
	key.SetInt64("value", 200)

	require.Equal(t, int64(200), key.AsInt64("value"), "Value for 'value' should be overwritten to 200")
}

func Test_STSKey_PanicMessages(t *testing.T) {
	key := NewSTSKey("namespace", "name")

	require.PanicsWithValue(t, "value missing for key 'missingKey'", func() {
		key.AsInt64("missingKey")
	}, "Should panic with correct message for missing key")

	key.SetString("wrongType", "stringValue")
	require.PanicsWithValue(t, "value for key 'wrongType' is not int64", func() {
		key.AsInt64("wrongType")
	}, "Should panic with correct message for type mismatch")
}
