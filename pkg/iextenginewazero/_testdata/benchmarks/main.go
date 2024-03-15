/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */

package main

import (
	ext "github.com/voedger/voedger/pkg/exttinygo"
)

const StorageTest = "sys.Test"
const StorageTest2 = "sys.Test2"

//export oneGetOneIntent5calls
func oneGetOneIntent5calls() {
	ext.MustGetValue(ext.KeyBuilder(ext.StorageEvent, ext.NullEntity))
	mail := ext.NewValue(ext.KeyBuilder(ext.StorageSendMail, ext.NullEntity))
	mail.PutString("from", "test@gmail.com")
}

//export oneGetNoIntents2calls
func oneGetNoIntents2calls() {
	ext.MustGetValue(ext.KeyBuilder(ext.StorageEvent, ext.NullEntity))
}

//export oneGetLongStr3calls
func oneGetLongStr3calls() {
	value := ext.MustGetValue(ext.KeyBuilder(StorageTest, ext.NullEntity))
	value.AsString("500c")
}

//export doNothing
func doNothing() {
}

//export oneKey1call
func oneKey1call() {
	ext.KeyBuilder(StorageTest2, ext.NullEntity)
}

func main() {
}
