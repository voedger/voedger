/*
  - Copyright (c) 2023-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/

package exttinygotests

import "github.com/voedger/voedger/pkg/exttinygo"

func initKeyBuilder() {
	exttinygo.KeyBuilderPutInt32 = func(k exttinygo.TKeyBuilder, name string, value int32) {
		currentCtx.keyBuilders[k].PutInt32(name, value)
	}
}
