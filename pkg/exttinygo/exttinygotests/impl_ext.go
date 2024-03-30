/*
  - Copyright (c) 2023-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/

package exttinygotests

import "github.com/voedger/voedger/pkg/exttinygo"

func initExt() {
	exttinygo.MustGetValue = func(key exttinygo.TKeyBuilder) (v exttinygo.TValue) {
		sv, err := currentCtx.io.MustExist(currentCtx.keyBuilders[key])
		if err != nil {
			panic(err)
		}
		v = exttinygo.TValue(len(currentCtx.values))
		currentCtx.values = append(currentCtx.values, sv)
		return v
	}
	exttinygo.QueryValue = func(key exttinygo.TKeyBuilder) (v exttinygo.TValue, ok bool) {
		sv, ok, err := currentCtx.io.CanExist(currentCtx.keyBuilders[key])
		if err != nil {
			panic(err)
		}
		if ok {
			v = exttinygo.TValue(len(currentCtx.values))
			currentCtx.values = append(currentCtx.values, sv)
			return v, true
		}
		return 0, false
	}
	exttinygo.NewValue = func(key exttinygo.TKeyBuilder) (v exttinygo.TIntent) {
		svb, err := currentCtx.io.NewValue(currentCtx.keyBuilders[key])
		if err != nil {
			panic(err)
		}
		v = exttinygo.TIntent(len(currentCtx.valueBuilders))
		currentCtx.valueBuilders = append(currentCtx.valueBuilders, svb)
		return v
	}
	exttinygo.UpdateValue = func(key exttinygo.TKeyBuilder, existingValue exttinygo.TValue) (v exttinygo.TIntent) {
		svb, err := currentCtx.io.UpdateValue(currentCtx.keyBuilders[key], currentCtx.values[existingValue])
		if err != nil {
			panic(err)
		}
		v = exttinygo.TIntent(len(currentCtx.valueBuilders))
		currentCtx.valueBuilders = append(currentCtx.valueBuilders, svb)
		return v
	}

}
