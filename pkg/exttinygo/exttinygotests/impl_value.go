/*
  - Copyright (c) 2023-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/

package exttinygotests

import "github.com/voedger/voedger/pkg/exttinygo"

func initValue() {
	exttinygo.ValueAsValue = func(v exttinygo.TValue, name string) (result exttinygo.TValue) {
		result = exttinygo.TValue(len(currentCtx.values))
		sv := currentCtx.values[v].AsValue(name)
		currentCtx.values = append(currentCtx.values, sv)
		return result
	}
	exttinygo.ValueLen = func(v exttinygo.TValue) int {
		return currentCtx.values[v].Length()

	}
	exttinygo.ValueGetAsValue = func(v exttinygo.TValue, index int) (result exttinygo.TValue) {
		result = exttinygo.TValue(len(currentCtx.values))
		sv := currentCtx.values[v].GetAsValue(index)
		currentCtx.values = append(currentCtx.values, sv)
		return result
	}
	exttinygo.ValueAsInt32 = func(v exttinygo.TValue, name string) int32 {
		return currentCtx.values[v].AsInt32(name)
	}
	exttinygo.ValueAsInt64 = func(v exttinygo.TValue, name string) int64 {
		return currentCtx.values[v].AsInt64(name)
	}
}
