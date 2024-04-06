/*
  - Copyright (c) 2023-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/

package exttinygo

import (
	"github.com/voedger/voedger/pkg/exttinygo/internal"
	"github.com/voedger/voedger/pkg/state/isafeapi"
)

func (kb TKeyBuilder) PutInt32(name string, value int32) {
	internal.StateAPI.KeyBuilderPutInt32(isafeapi.TKeyBuilder(kb), name, value)
}

func (kb TKeyBuilder) PutInt64(name string, value int64) {
	internal.StateAPI.KeyBuilderPutInt64(isafeapi.TKeyBuilder(kb), name, value)
}

func (kb TKeyBuilder) PutFloat32(name string, value float32) {
	internal.StateAPI.KeyBuilderPutFloat32(isafeapi.TKeyBuilder(kb), name, value)
}

func (kb TKeyBuilder) PutFloat64(name string, value float64) {
	internal.StateAPI.KeyBuilderPutFloat64(isafeapi.TKeyBuilder(kb), name, value)
}

func (kb TKeyBuilder) PutString(name string, value string) {
	internal.StateAPI.KeyBuilderPutString(isafeapi.TKeyBuilder(kb), name, value)
}

func (kb TKeyBuilder) PutBytes(name string, value []byte) {
	internal.StateAPI.KeyBuilderPutBytes(isafeapi.TKeyBuilder(kb), name, value)
}

func (kb TKeyBuilder) PutQName(name string, value QName) {
	internal.StateAPI.KeyBuilderPutQName(isafeapi.TKeyBuilder(kb), name, isafeapi.QName(value))
}

func (kb TKeyBuilder) PutBool(name string, value bool) {
	internal.StateAPI.KeyBuilderPutBool(isafeapi.TKeyBuilder(kb), name, value)
}
