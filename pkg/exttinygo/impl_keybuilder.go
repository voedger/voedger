/*
  - Copyright (c) 2023-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/

package exttinygo

import (
	"github.com/voedger/voedger/pkg/exttinygo/internal"
	"github.com/voedger/voedger/pkg/state/isafestate"
)

func (kb TKeyBuilder) PutInt32(name string, value int32) {
	internal.State.KeyBuilderPutInt32(isafestate.TKeyBuilder(kb), name, value)
}

func (kb TKeyBuilder) PutInt64(name string, value int64) {
	internal.State.KeyBuilderPutInt64(isafestate.TKeyBuilder(kb), name, value)
}

func (kb TKeyBuilder) PutFloat32(name string, value float32) {
	internal.State.KeyBuilderPutFloat32(isafestate.TKeyBuilder(kb), name, value)
}

func (kb TKeyBuilder) PutFloat64(name string, value float64) {
	internal.State.KeyBuilderPutFloat64(isafestate.TKeyBuilder(kb), name, value)
}

func (kb TKeyBuilder) PutString(name string, value string) {
	internal.State.KeyBuilderPutString(isafestate.TKeyBuilder(kb), name, value)
}

func (kb TKeyBuilder) PutBytes(name string, value []byte) {
	internal.State.KeyBuilderPutBytes(isafestate.TKeyBuilder(kb), name, value)
}

func (kb TKeyBuilder) PutQName(name string, value QName) {
	internal.State.KeyBuilderPutQName(isafestate.TKeyBuilder(kb), name, isafestate.QName(value))
}

func (kb TKeyBuilder) PutBool(name string, value bool) {
	internal.State.KeyBuilderPutBool(isafestate.TKeyBuilder(kb), name, value)
}
