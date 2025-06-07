/*
  - Copyright (c) 2023-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/

package exttinygo

import (
	"github.com/voedger/voedger/pkg/exttinygo/internal"
	safe "github.com/voedger/voedger/pkg/state/isafestateapi"
)

func (kb TKeyBuilder) PutInt32(name string, value int32) {
	internal.SafeStateAPI.KeyBuilderPutInt32(safe.TKeyBuilder(kb), name, value)
}

func (kb TKeyBuilder) PutInt64(name string, value int64) {
	internal.SafeStateAPI.KeyBuilderPutInt64(safe.TKeyBuilder(kb), name, value)
}

func (kb TKeyBuilder) PutRecordID(name string, value int64) {
	internal.SafeStateAPI.KeyBuilderPutRecordID(safe.TKeyBuilder(kb), name, value)
}

func (kb TKeyBuilder) PutFloat32(name string, value float32) {
	internal.SafeStateAPI.KeyBuilderPutFloat32(safe.TKeyBuilder(kb), name, value)
}

func (kb TKeyBuilder) PutFloat64(name string, value float64) {
	internal.SafeStateAPI.KeyBuilderPutFloat64(safe.TKeyBuilder(kb), name, value)
}

func (kb TKeyBuilder) PutString(name string, value string) {
	internal.SafeStateAPI.KeyBuilderPutString(safe.TKeyBuilder(kb), name, value)
}

func (kb TKeyBuilder) PutBytes(name string, value []byte) {
	internal.SafeStateAPI.KeyBuilderPutBytes(safe.TKeyBuilder(kb), name, value)
}

func (kb TKeyBuilder) PutQName(name string, value QName) {
	internal.SafeStateAPI.KeyBuilderPutQName(safe.TKeyBuilder(kb), name, safe.QName(value))
}

func (kb TKeyBuilder) PutBool(name string, value bool) {
	internal.SafeStateAPI.KeyBuilderPutBool(safe.TKeyBuilder(kb), name, value)
}
