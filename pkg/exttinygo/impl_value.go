/*
  - Copyright (c) 2023-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/

package exttinygo

import (
	"github.com/voedger/voedger/pkg/exttinygo/internal"
	"github.com/voedger/voedger/pkg/state/isafeapi"
)

func (v TValue) AsInt32(name string) int32 {
	return internal.StateAPI.ValueAsInt32(isafeapi.TValue(v), name)
}

func (v TValue) AsInt64(name string) int64 {
	return internal.StateAPI.ValueAsInt64(isafeapi.TValue(v), name)
}

func (v TValue) AsFloat32(name string) float32 {
	return internal.StateAPI.ValueAsFloat32(isafeapi.TValue(v), name)
}

func (v TValue) AsFloat64(name string) float64 {
	return internal.StateAPI.ValueAsFloat64(isafeapi.TValue(v), name)
}

func (v TValue) AsString(name string) string {
	return internal.StateAPI.ValueAsString(isafeapi.TValue(v), name)
}

func (v TValue) AsBytes(name string) []byte {
	return internal.StateAPI.ValueAsBytes(isafeapi.TValue(v), name)
}

func (v TValue) AsQName(name string) QName {
	return QName(internal.StateAPI.ValueAsQName(isafeapi.TValue(v), name))
}

func (v TValue) AsBool(name string) bool {
	return internal.StateAPI.ValueAsBool(isafeapi.TValue(v), name)
}

func (v TValue) AsValue(name string) TValue {
	return TValue(internal.StateAPI.ValueAsValue(isafeapi.TValue(v), name))
}

func (v TValue) Len() int {
	return internal.StateAPI.ValueLen(isafeapi.TValue(v))
}

func (v TValue) GetAsValue(index int) TValue {
	return TValue(internal.StateAPI.ValueGetAsValue(isafeapi.TValue(v), index))
}

func (v TValue) GetAsInt32(index int) int32 {
	return internal.StateAPI.ValueGetAsInt32(isafeapi.TValue(v), index)
}

func (v TValue) GetAsInt64(index int) int64 {
	return internal.StateAPI.ValueGetAsInt64(isafeapi.TValue(v), index)
}

func (v TValue) GetAsFloat32(index int) float32 {
	return internal.StateAPI.ValueGetAsFloat32(isafeapi.TValue(v), index)
}

func (v TValue) GetAsFloat64(index int) float64 {
	return internal.StateAPI.ValueGetAsFloat64(isafeapi.TValue(v), index)
}

func (v TValue) GetAsBytes(index int) []byte {
	return internal.StateAPI.ValueGetAsBytes(isafeapi.TValue(v), index)
}

func (v TValue) GetAsQName(index int) QName {
	return QName(internal.StateAPI.ValueGetAsQName(isafeapi.TValue(v), index))
}

func (v TValue) GetAsBool(index int) bool {
	return internal.StateAPI.ValueGetAsBool(isafeapi.TValue(v), index)
}

func (v TValue) GetAsString(index int) string {
	return internal.StateAPI.ValueGetAsString(isafeapi.TValue(v), index)
}
