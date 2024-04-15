/*
  - Copyright (c) 2023-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/

package exttinygo

import (
	"github.com/voedger/voedger/pkg/exttinygo/internal"
	safe "github.com/voedger/voedger/pkg/state/isafestateapi"
)

func (k TKey) AsInt32(name string) int32 {
	return internal.SafeStateAPI.KeyAsInt32(safe.TKey(k), name)
}

func (k TKey) AsInt64(name string) int64 {
	return internal.SafeStateAPI.KeyAsInt64(safe.TKey(k), name)
}

func (k TKey) AsFloat32(name string) float32 {
	return internal.SafeStateAPI.KeyAsFloat32(safe.TKey(k), name)
}

func (k TKey) AsFloat64(name string) float64 {
	return internal.SafeStateAPI.KeyAsFloat64(safe.TKey(k), name)
}

func (k TKey) AsBytes(name string) []byte {
	return internal.SafeStateAPI.KeyAsBytes(safe.TKey(k), name)
}

func (k TKey) AsString(name string) string {
	return internal.SafeStateAPI.KeyAsString(safe.TKey(k), name)
}

func (k TKey) AsQName(name string) QName {
	return QName(internal.SafeStateAPI.KeyAsQName(safe.TKey(k), name))
}

func (k TKey) AsBool(name string) bool {
	return internal.SafeStateAPI.KeyAsBool(safe.TKey(k), name)
}
