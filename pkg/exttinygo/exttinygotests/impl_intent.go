/*
  - Copyright (c) 2023-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/

package exttinygotests

import "github.com/voedger/voedger/pkg/exttinygo"

func initIntent() {
	exttinygo.IntentPutInt64 = func(v exttinygo.TIntent, name string, value int64) {
		currentCtx.valueBuilders[v].PutInt64(name, value)
	}
}
