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
	return internal.State.ValueAsInt32(isafeapi.TValue(v), name)
}

func (v TValue) AsInt64(name string) int64 {
	return internal.State.ValueAsInt64(isafeapi.TValue(v), name)
}

func (v TValue) AsFloat32(name string) float32 {
	return internal.State.ValueAsFloat32(isafeapi.TValue(v), name)
}

func (v TValue) AsFloat64(name string) float64 {
	return internal.State.ValueAsFloat64(isafeapi.TValue(v), name)
}

func (v TValue) AsString(name string) string {
	return internal.State.ValueAsString(isafeapi.TValue(v), name)
}

func (v TValue) AsBytes(name string) []byte {
	return internal.State.ValueAsBytes(isafeapi.TValue(v), name)
}

func (v TValue) AsQName(name string) QName {
	return QName(internal.State.ValueAsQName(isafeapi.TValue(v), name))
}

func (v TValue) AsBool(name string) bool {
	return internal.State.ValueAsBool(isafeapi.TValue(v), name)
}

func (v TValue) AsValue(name string) TValue {
	return TValue(internal.State.ValueAsValue(isafeapi.TValue(v), name))
}

func (v TValue) Len() int {
	return internal.State.ValueLen(isafeapi.TValue(v))
}

func (v TValue) GetAsValue(index int) TValue {
	return TValue(internal.State.ValueGetAsValue(isafeapi.TValue(v), index))
}

func (v TValue) GetAsInt32(index int) int32 {
	return internal.State.ValueGetAsInt32(isafeapi.TValue(v), index)
}

func (v TValue) GetAsInt64(index int) int64 {
	return internal.State.ValueGetAsInt64(isafeapi.TValue(v), index)
}

func (v TValue) GetAsFloat32(index int) float32 {
	return internal.State.ValueGetAsFloat32(isafeapi.TValue(v), index)
}

func (v TValue) GetAsFloat64(index int) float64 {
	return internal.State.ValueGetAsFloat64(isafeapi.TValue(v), index)
}

func (v TValue) GetAsBytes(index int) []byte {
	return internal.State.ValueGetAsBytes(isafeapi.TValue(v), index)
}

func (v TValue) GetAsQName(index int) QName {
	return QName(internal.State.ValueGetAsQName(isafeapi.TValue(v), index))
}

func (v TValue) GetAsBool(index int) bool {
	return internal.State.ValueGetAsBool(isafeapi.TValue(v), index)
}

func (v TValue) GetAsString(index int) string {
	return internal.State.ValueGetAsString(isafeapi.TValue(v), index)
}
