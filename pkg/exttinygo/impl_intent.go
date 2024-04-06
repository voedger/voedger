/*
  - Copyright (c) 2023-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/

package exttinygo

import (
	"github.com/voedger/voedger/pkg/exttinygo/internal"
	"github.com/voedger/voedger/pkg/state/isafeapi"
)

func (i TIntent) PutInt32(name string, value int32) {
	internal.StateAPI.IntentPutInt32(isafeapi.TIntent(i), name, value)
}

func (i TIntent) PutInt64(name string, value int64) {
	internal.StateAPI.IntentPutInt64(isafeapi.TIntent(i), name, value)
}

func (i TIntent) PutFloat32(name string, value float32) {
	internal.StateAPI.IntentPutFloat32(isafeapi.TIntent(i), name, value)
}

func (i TIntent) PutFloat64(name string, value float64) {
	internal.StateAPI.IntentPutFloat64(isafeapi.TIntent(i), name, value)
}

func (i TIntent) PutString(name string, value string) {
	internal.StateAPI.IntentPutString(isafeapi.TIntent(i), name, value)
}

func (i TIntent) PutBytes(name string, value []byte) {
	internal.StateAPI.IntentPutBytes(isafeapi.TIntent(i), name, value)
}

func (i TIntent) PutQName(name string, value QName) {
	internal.StateAPI.IntentPutQName(isafeapi.TIntent(i), name, isafeapi.QName(value))
}

func (i TIntent) PutBool(name string, value bool) {
	internal.StateAPI.IntentPutBool(isafeapi.TIntent(i), name, value)
}
