/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */

package main

import (
	ext "github.com/voedger/voedger/pkg/exttinygo"
	require "github.com/voedger/voedger/pkg/exttinygo/require"
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
		qname:   ext.QName{FullPkgName: "github.com/voedger/testpkg1", Entity: "e1"},
		boolean: true,
	},
	expVal{
		i32:     2,
		i64:     12,
		f32:     2.1,
		f64:     2.01,
		str:     "key2",
		bytes:   []byte{2, 2, 3},
		qname:   ext.QName{FullPkgName: "github.com/voedger/testpkg1", Entity: "e2"},
		boolean: true,
	},
	expVal{
		i32:     3,
		i64:     13,
		f32:     3.1,
		f64:     3.01,
		str:     "key3",
		bytes:   []byte{3, 2, 3},
		qname:   ext.QName{FullPkgName: "github.com/voedger/testpkg1", Entity: "e3"},
		boolean: true,
	},
}

var expectedValues []expVal = []expVal{
	expVal{
		i32:     101,
		i64:     1001,
		f32:     1.001,
		f64:     1.0001,
		str:     "value1",
		bytes:   []byte{3, 2, 1},
		qname:   ext.QName{FullPkgName: "github.com/voedger/testpkg1", Entity: "ee1"},
		boolean: false,
	},
	expVal{
		i32:     102,
		i64:     1002,
		f32:     2.001,
		f64:     2.0001,
		str:     "value2",
		bytes:   []byte{3, 2, 1},
		qname:   ext.QName{FullPkgName: "github.com/voedger/testpkg1", Entity: "ee2"},
		boolean: false,
	},
	expVal{
		i32:     103,
		i64:     1003,
		f32:     3.001,
		f64:     3.0001,
		str:     "value3",
		bytes:   []byte{3, 2, 1},
		qname:   ext.QName{FullPkgName: "github.com/voedger/testpkg1", Entity: "e33"},
		boolean: false,
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

		require.EqualInt32(expectedValues[i].i32, value.AsInt32("i32"))
		require.EqualInt64(expectedValues[i].i64, value.AsInt64("i64"))
		require.EqualFloat32(expectedValues[i].f32, value.AsFloat32("f32"))
		require.EqualFloat64(expectedValues[i].f64, value.AsFloat64("f64"))
		require.EqualString(expectedValues[i].str, value.AsString("str"))
		require.EqualBytes(expectedValues[i].bytes, value.AsBytes("bytes"))
		require.EqualQName(expectedValues[i].qname, value.AsQName("qname"))
		require.EqualBool(expectedValues[i].boolean, value.AsBool("bool"))

		i++
	})
	require.EqualInt32(3, int32(i))

}

//export asBytes
func asBytes() {
	key := ext.KeyBuilder("sys.Test", ext.NullEntity)
	value := ext.MustGetValue(key)
	bytes := value.AsBytes("bytes")
	require.EqualInt32(2000000, int32(len(bytes)))
}

var mybytes = make([]byte, 5)

//export testNoAllocs
func testNoAllocs() {

	// KeyBuilder
	kb := ext.KeyBuilder(ext.StorageEvent, ext.NullEntity)
	kb.PutString("somekey", "somevalue")
	kb.PutBytes("somebytes", mybytes)

	// QueryValue
	event, exists := ext.QueryValue(kb)
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
	require.EqualQName(ext.QName{FullPkgName: "github.com/voedger/testpkg1", Entity: "test"}, value.GetAsQName(3))

	require.EqualBool(true, len(bytes) == 1024)

	// Read
	testRead()

	// NewValue
	mail := ext.NewValue(ext.KeyBuilder(ext.StorageSendMail, ext.NullEntity))
	mail.PutString("from", "test@gmail.com")
	mail.PutInt32("port", 668)
	mail.PutBytes("key", mybytes)
	mail.PutQName("qname", ext.QName{FullPkgName: "github.com/voedger/testpkg1", Entity: "test"})

	// UpdateValue
	updatedValue := ext.UpdateValue(kb, event)
	updatedValue.PutInt32("offs", event.AsInt32("offs")+1)

}

//export testQueryValue
func testQueryValue() {
	kb := ext.KeyBuilder(ext.StorageRecord, ext.NullEntity)
	_, exists := ext.QueryValue(kb)
	require.EqualBool(false, exists)
}

//export keyPutQName
func keyPutQName() {
	kb := ext.KeyBuilder("sys.TestQName", ext.NullEntity)
	kb.PutQName("qname", ext.QName{FullPkgName: "github.com/voedger/testpkg1", Entity: "test"})
	_ = ext.MustGetValue(kb)
}

//export ProjectorTestStorageWLog
func ProjectorTestStorageWLog() {
	event := ext.MustGetValue(ext.KeyBuilder(ext.StorageEvent, ext.NullEntity))
	arg := event.AsValue("ArgumentObject")
	offs := arg.AsInt64("Offset")
	count := arg.AsInt64("Count")

	kb := ext.KeyBuilder(ext.StorageWLog, ext.NullEntity)
	kb.PutInt64("Offset", offs)

	// read 1 item
	wlogEntry := ext.MustGetValue(kb)
	qname := wlogEntry.AsQName("QName")

	// read range
	kb = ext.KeyBuilder(ext.StorageWLog, ext.NullEntity)
	kb.PutInt64("Offset", offs)
	kb.PutInt64("Count", count)

	var readValues int32
	ext.ReadValues(kb, func(key ext.TKey, value ext.TValue) {
		readValues++
	})

	result := ext.NewValue(ext.KeyBuilder(ext.StorageView, "github.com/org/app/packages/mypkg.Results"))
	result.PutInt32("IntVal", readValues)
	result.PutQName("QNameVal", qname)
}

//export ProjectorTestStorages
func ProjectorTestStorages() {
	secret := ext.KeyBuilder(ext.StorageAppSecret, ext.NullEntity)
	secret.PutString("Secret", "smtpPassword")
	smtpPassword := ext.MustGetValue(secret)

	var userName string
	http := ext.KeyBuilder(ext.StorageHTTP, ext.NullEntity)
	http.PutString("Method", "GET")
	http.PutString("Url", "/getUsername")
	http.PutInt64("HTTPClientTimeoutMilliseconds", 1000)
	http.PutString("Header", "my-header: my-value")
	http.PutString("Header", "Content-type: application/json")
	ext.ReadValues(http, func(key ext.TKey, value ext.TValue) {
		if value.AsInt32("StatusCode") == 200 {
			userName = string(value.AsBytes("Body"))
		}
	})

	email := ext.KeyBuilder(ext.StorageSendMail, ext.NullEntity)
	email.PutString("Host", "smtp.gmail.com")
	email.PutInt32("Port", 587)
	email.PutString("From", "no-reply@gmail.com")
	email.PutString("To", "email@gmail.com")
	email.PutString("Subject", "Test")
	email.PutString("Username", userName)
	email.PutString("Password", smtpPassword.AsString(""))
	email.PutString("Body", "TheBody")

	_ = ext.NewValue(email)
}

//export CommandTestStorages
func CommandTestStorages() {
	arg := ext.MustGetValue(ext.KeyBuilder(ext.StorageCommandContext, ext.NullEntity)).AsValue("ArgumentObject")
	idToRead := arg.AsInt64("IdToRead")

	kb := ext.KeyBuilder(ext.StorageRecord, ext.NullEntity)
	kb.PutInt64("ID", idToRead)
	rec := ext.MustGetValue(kb)

	kb = ext.KeyBuilder(ext.StorageRequestSubject, ext.NullEntity)
	principal := ext.MustGetValue(kb)
	wsid := principal.AsInt64("ProfileWSID")
	kind := principal.AsInt32("Kind")
	name := principal.AsString("Name")
	token := principal.AsString("Token")

	result := ext.NewValue(ext.KeyBuilder(ext.StorageResult, ext.NullEntity))
	result.PutInt32("ReadValue", rec.AsInt32("Value"))
	result.PutInt64("ReadWSID", wsid)
	result.PutInt32("ReadKind", kind)
	result.PutString("ReadName", name)
	result.PutString("ReadToken", token)

	cud := ext.NewValue(ext.KeyBuilder(ext.StorageRecord, "github.com/org/app/packages/mypkg.Doc1"))
	cud.PutInt32("Value", 43)
}

func main() {

}
