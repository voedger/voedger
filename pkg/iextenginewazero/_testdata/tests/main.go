/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */

package main

import (
	ext "github.com/heeus/extensions-tinygo"
	require "github.com/heeus/extensions-tinygo/require"
)

type expVal struct {
	i32     int32
	i64     int64
	f32     float32
	f64     float64
	str     string
	bytes   []byte
	qname   ext.QName
	boolean bool
}

var expectedKeys []expVal = []expVal{
	expVal{
		i32:     1,
		i64:     11,
		f32:     1.1,
		f64:     1.01,
		str:     "key1",
		bytes:   []byte{1, 2, 3},
		qname:   ext.QName{Pkg: "keypkg", Entity: "e1"},
		boolean: true,
	},
	expVal{
		i32:     2,
		i64:     12,
		f32:     2.1,
		f64:     2.01,
		str:     "key2",
		bytes:   []byte{2, 2, 3},
		qname:   ext.QName{Pkg: "keypkg", Entity: "e2"},
		boolean: true,
	},
	expVal{
		i32:     3,
		i64:     13,
		f32:     3.1,
		f64:     3.01,
		str:     "key3",
		bytes:   []byte{3, 2, 3},
		qname:   ext.QName{Pkg: "keypkg", Entity: "e3"},
		boolean: true,
	},
}

var i int

//export testRead
func testRead() {
	key := ext.KeyBuilder("sys.Test", ext.NullEntity)
	i = 0
	_ = expectedKeys[0].i32
	ext.ReadValues(key, func(key ext.TKey, value ext.TValue) {
		require.EqualInt32(expectedKeys[i].i32, key.AsInt32("i32"))
		require.EqualInt64(expectedKeys[i].i64, key.AsInt64("i64"))
		require.EqualFloat32(expectedKeys[i].f32, key.AsFloat32("f32"))
		require.EqualFloat64(expectedKeys[i].f64, key.AsFloat64("f64"))
		require.EqualString(expectedKeys[i].str, key.AsString("str"))
		require.EqualBytes(expectedKeys[i].bytes, key.AsBytes("bytes"))
		require.EqualQName(expectedKeys[i].qname, key.AsQName("qname"))
		require.EqualBool(expectedKeys[i].boolean, key.AsBool("bool"))
		i++
	})
	require.EqualInt32(3, int32(i))

}

//export asBytes
func asBytes() {
	key := ext.KeyBuilder("sys.Test", ext.NullEntity)
	value := ext.MustGetValue(key)
	bytes := value.AsBytes("bytes")
	require.EqualBool(true, len(bytes) == 2000000)
}

var mybytes = make([]byte, 5)

//export testNoAllocs
func testNoAllocs() {

	// KeyBuilder
	kb := ext.KeyBuilder(ext.StorageEvent, ext.NullEntity)
	kb.PutString("somekey", "somevalue")
	kb.PutBytes("somebytes", mybytes)

	// QueryValue
	exists, event := ext.QueryValue(kb)
	require.EqualBool(true, exists)
	require.EqualInt32(int32(12345), event.AsInt32("offs"))

	arg := event.AsValue("arg")
	require.EqualString("email@user.com", arg.AsString("UserEmail"))

	// GetValue + GetAs*
	kb2 := ext.KeyBuilder("sys.Test3", ext.NullEntity)
	value := ext.MustGetValue(kb2)
	require.EqualInt32(int32(123), value.GetAsInt32(0))
	require.EqualString("test string", value.GetAsString(1))
	bytes := value.GetAsBytes(2)

	require.EqualBool(true, len(bytes) == 1024)

	// Read
	testRead()

	// NewValue
	mail := ext.NewValue(ext.KeyBuilder(ext.StorageSendmail, ext.NullEntity))
	mail.PutString("from", "test@gmail.com")
	mail.PutInt32("port", 668)
	mail.PutBytes("key", mybytes)

	// UpdateValue
	updatedValue := ext.UpdateValue(kb, event)
	updatedValue.PutInt32("offs", event.AsInt32("offs")+1)

}

//export testQueryValue
func testQueryValue() {
	kb := ext.KeyBuilder(ext.StorageRecords, ext.NullEntity)
	exists, _ := ext.QueryValue(kb)
	require.EqualBool(false, exists)
}

func main() {

}
