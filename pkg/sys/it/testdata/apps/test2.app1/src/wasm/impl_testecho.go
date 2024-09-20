/*
 * Copyright (c) 2023-present unTill Software Development Group B.V.
 * @author Michael Saigachenko
 */

package main

import (
	ext "github.com/voedger/voedger/pkg/exttinygo"
)

func main() {}

//export TestEcho
func TestEcho() {
	arg := ext.MustGetValue(ext.KeyBuilder(ext.StorageQueryContext, ext.NullEntity)).AsValue(cmdContext_Argument)
	str := arg.AsString(field_Str)

	result := ext.NewValue(ext.KeyBuilder(ext.StorageResult, ext.NullEntity))
	result.PutString(field_Res, "hello, "+str)
}

//export TestCmdEcho
func TestCmdEcho() {
	arg := ext.MustGetValue(ext.KeyBuilder(ext.StorageCommandContext, ext.NullEntity)).AsValue(cmdContext_Argument)
	str := arg.AsString(field_Str)

	result := ext.NewValue(ext.KeyBuilder(ext.StorageResult, ext.NullEntity))
	result.PutString(field_Res, "hello, "+str)
}
