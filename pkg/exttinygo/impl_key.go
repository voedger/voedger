/*
  - Copyright (c) 2023-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/

package exttinygo

import (
	"github.com/voedger/voedger/pkg/exttinygo/internal"
	safe "github.com/voedger/voedger/pkg/state/isafestate"
)

func (k TKey) AsInt32(name string) int32 {
	return internal.State.KeyAsInt32(safe.TKey(k), name)
}

func (k TKey) AsInt64(name string) int64 {
	return internal.State.KeyAsInt64(safe.TKey(k), name)
}

func (k TKey) AsFloat32(name string) float32 {
	return internal.State.KeyAsFloat32(safe.TKey(k), name)
}

func (k TKey) AsFloat64(name string) float64 {
	return internal.State.KeyAsFloat64(safe.TKey(k), name)
}

func (k TKey) AsBytes(name string) []byte {
	return internal.State.KeyAsBytes(safe.TKey(k), name)
}

func (k TKey) AsString(name string) string {
	return internal.State.KeyAsString(safe.TKey(k), name)
}

func (k TKey) AsQName(name string) QName {
	return QName(internal.State.KeyAsQName(safe.TKey(k), name))
}

func (k TKey) AsBool(name string) bool {
	return internal.State.KeyAsBool(safe.TKey(k), name)
}
