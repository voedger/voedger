/*
  - Copyright (c) 2023-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/

package exttinygo

import (
	"github.com/voedger/voedger/pkg/exttinygo/internal"
	safe "github.com/voedger/voedger/pkg/state/isafestateapi"
)

func (i TIntent) PutInt32(name string, value int32) {
	internal.SafeStateAPI.IntentPutInt32(safe.TIntent(i), name, value)
}

func (i TIntent) PutInt64(name string, value int64) {
	internal.SafeStateAPI.IntentPutInt64(safe.TIntent(i), name, value)
}

func (i TIntent) PutFloat32(name string, value float32) {
	internal.SafeStateAPI.IntentPutFloat32(safe.TIntent(i), name, value)
}

func (i TIntent) PutFloat64(name string, value float64) {
	internal.SafeStateAPI.IntentPutFloat64(safe.TIntent(i), name, value)
}

func (i TIntent) PutString(name string, value string) {
	internal.SafeStateAPI.IntentPutString(safe.TIntent(i), name, value)
}

func (i TIntent) PutBytes(name string, value []byte) {
	internal.SafeStateAPI.IntentPutBytes(safe.TIntent(i), name, value)
}

func (i TIntent) PutQName(name string, value QName) {
	internal.SafeStateAPI.IntentPutQName(safe.TIntent(i), name, safe.QName(value))
}

func (i TIntent) PutBool(name string, value bool) {
	internal.SafeStateAPI.IntentPutBool(safe.TIntent(i), name, value)
}
