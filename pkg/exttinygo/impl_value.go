/*
  - Copyright (c) 2023-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/

package exttinygo

import (
	"github.com/voedger/voedger/pkg/exttinygo/internal"
	safe "github.com/voedger/voedger/pkg/state/isafestateapi"
)

func (v TValue) AsInt32(name string) int32 {
	return internal.SafeStateAPI.ValueAsInt32(safe.TValue(v), name)
}

func (v TValue) AsInt64(name string) int64 {
	return internal.SafeStateAPI.ValueAsInt64(safe.TValue(v), name)
}

func (v TValue) AsFloat32(name string) float32 {
	return internal.SafeStateAPI.ValueAsFloat32(safe.TValue(v), name)
}

func (v TValue) AsFloat64(name string) float64 {
	return internal.SafeStateAPI.ValueAsFloat64(safe.TValue(v), name)
}

func (v TValue) AsString(name string) string {
	return internal.SafeStateAPI.ValueAsString(safe.TValue(v), name)
}

func (v TValue) AsBytes(name string) []byte {
	return internal.SafeStateAPI.ValueAsBytes(safe.TValue(v), name)
}

func (v TValue) AsQName(name string) QName {
	return QName(internal.SafeStateAPI.ValueAsQName(safe.TValue(v), name))
}

func (v TValue) AsBool(name string) bool {
	return internal.SafeStateAPI.ValueAsBool(safe.TValue(v), name)
}

func (v TValue) AsValue(name string) TValue {
	return TValue(internal.SafeStateAPI.ValueAsValue(safe.TValue(v), name))
}

func (v TValue) Len() int {
	return internal.SafeStateAPI.ValueLen(safe.TValue(v))
}

func (v TValue) GetAsValue(index int) TValue {
	return TValue(internal.SafeStateAPI.ValueGetAsValue(safe.TValue(v), index))
}

func (v TValue) GetAsInt32(index int) int32 {
	return internal.SafeStateAPI.ValueGetAsInt32(safe.TValue(v), index)
}

func (v TValue) GetAsInt64(index int) int64 {
	return internal.SafeStateAPI.ValueGetAsInt64(safe.TValue(v), index)
}

func (v TValue) GetAsFloat32(index int) float32 {
	return internal.SafeStateAPI.ValueGetAsFloat32(safe.TValue(v), index)
}

func (v TValue) GetAsFloat64(index int) float64 {
	return internal.SafeStateAPI.ValueGetAsFloat64(safe.TValue(v), index)
}

func (v TValue) GetAsBytes(index int) []byte {
	return internal.SafeStateAPI.ValueGetAsBytes(safe.TValue(v), index)
}

func (v TValue) GetAsQName(index int) QName {
	return QName(internal.SafeStateAPI.ValueGetAsQName(safe.TValue(v), index))
}

func (v TValue) GetAsBool(index int) bool {
	return internal.SafeStateAPI.ValueGetAsBool(safe.TValue(v), index)
}

func (v TValue) GetAsString(index int) string {
	return internal.SafeStateAPI.ValueGetAsString(safe.TValue(v), index)
}
