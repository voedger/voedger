/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */

package main

import (
	"time"

	ext "github.com/voedger/voedger/pkg/exttinygo"
)

//export incorrectStorageQname
func incorrectStorageQname() {
	ext.MustGetValue(ext.KeyBuilder("foo", ext.NullEntity))
}

//export incorrectEntityQname
func incorrectEntityQname() {
	ext.MustGetValue(ext.KeyBuilder("foo.FakeStorage", "abc"))
}

//export unsupportedStorage
func unsupportedStorage() {
	ext.MustGetValue(ext.KeyBuilder("foo.FakeStorage", ext.NullEntity))
}

//export incorrectKeyBuilder
func incorrectKeyBuilder() {
	ext.TKeyBuilder(123).PutInt32("a", 123)
}

//export mustExistIncorrectKey
func mustExistIncorrectKey() {
	ext.MustGetValue(ext.TKeyBuilder(123))
}

//export canExistIncorrectKey
func canExistIncorrectKey() {
	ext.MustGetValue(ext.TKeyBuilder(123))
}

//export mustExist
func mustExist() {
	ext.MustGetValue(ext.KeyBuilder(ext.StorageRecord, ext.NullEntity))
}

//export readIncorrectKeyBuilder
func readIncorrectKeyBuilder() {
	ext.ReadValues(ext.TKeyBuilder(123), func(ext.TKey, ext.TValue) {})
}

//export incorrectKey
func incorrectKey() {
	ext.TKey(123).AsString("boom")
}

//export incorrectValue
func incorrectValue() {
	ext.TValue(123).AsString("boom")
}

//export incorrectValue2
func incorrectValue2() {
	ext.TValue(123).GetAsString(1)
}

//export incorrectValue3
func incorrectValue3() {
	ext.TValue(123).Len()
}

//export incorrectKeyBuilderOnNewValue
func incorrectKeyBuilderOnNewValue() {
	ext.NewValue(ext.TKeyBuilder(123))
}

//export incorrectKeyBuilderOnUpdateValue
func incorrectKeyBuilderOnUpdateValue() {
	ext.UpdateValue(ext.TKeyBuilder(123), ext.TValue(123))
}

//export incorrectValueOnUpdateValue
func incorrectValueOnUpdateValue() {
	ext.UpdateValue(ext.KeyBuilder(ext.StorageEvent, ext.NullEntity), ext.TValue(123))
}

//export incorrectIntentId
func incorrectIntentId() {
	ext.TIntent(123).PutString("fake", "boom")
}

//export readPanic
func readPanic() {
	key := ext.KeyBuilder("sys.Test", ext.NullEntity)
	ext.ReadValues(key, func(ext.TKey, ext.TValue) {
		ext.TValue(123).Len()
	})
}

//export readError
func readError() {
	ext.ReadValues(ext.KeyBuilder("sys.IoErrorStorage", ext.NullEntity), func(ext.TKey, ext.TValue) {})
}

//export getError
func getError() {
	ext.MustGetValue(ext.KeyBuilder("sys.IoErrorStorage", ext.NullEntity))
}

//export queryError
func queryError() {
	ext.QueryValue(ext.KeyBuilder("sys.IoErrorStorage", ext.NullEntity))
}

//export newValueError
func newValueError() {
	ext.NewValue(ext.KeyBuilder("sys.IoErrorStorage", ext.NullEntity))
}

//export updateValueError
func updateValueError() {
	event := ext.MustGetValue(ext.KeyBuilder(ext.StorageEvent, ext.NullEntity))
	ext.UpdateValue(ext.KeyBuilder("sys.IoErrorStorage", ext.NullEntity), event)
}

//export asStringMemoryOverflow
func asStringMemoryOverflow() {
	key := ext.KeyBuilder("sys.Test", ext.NullEntity)
	value := ext.MustGetValue(key)
	value.AsBytes("bytes")
}

//export wrongFieldName
func wrongFieldName() {
	key := ext.KeyBuilder(ext.StorageView, "mypkg.TestView")
	key.PutInt32("wrong", 1)
	key.PutInt32("cc", 1)
	ext.MustGetValue(key)
}

//export undefinedPackage
func undefinedPackage() {
	key := ext.KeyBuilder(ext.StorageView, "github.com/company/pkg.Undefined")
	ext.MustGetValue(key)
}

//export TestPanic
func TestPanic() {
	panic("goodbye, world")
}

//export TestSignExtensionsFuncs
func TestSignExtensionsFuncs() {
	if time.Now().Year() < 2024 {
		panic("unexpected year")
	}
}

func main() {
}
