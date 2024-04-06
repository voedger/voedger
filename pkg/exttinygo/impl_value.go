/*
  - Copyright (c) 2023-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/

package exttinygo

import (
	"github.com/voedger/voedger/pkg/exttinygo/internal"
	"github.com/voedger/voedger/pkg/state/isafestate"
)

func (v TValue) AsInt32(name string) int32 {
	return internal.State.ValueAsInt32(isafestate.TValue(v), name)
}

func (v TValue) AsInt64(name string) int64 {
	return internal.State.ValueAsInt64(isafestate.TValue(v), name)
}

func (v TValue) AsFloat32(name string) float32 {
	return internal.State.ValueAsFloat32(isafestate.TValue(v), name)
}

func (v TValue) AsFloat64(name string) float64 {
	return internal.State.ValueAsFloat64(isafestate.TValue(v), name)
}

func (v TValue) AsString(name string) string {
	return internal.State.ValueAsString(isafestate.TValue(v), name)
}

func (v TValue) AsBytes(name string) []byte {
	return internal.State.ValueAsBytes(isafestate.TValue(v), name)
}

func (v TValue) AsQName(name string) QName {
	return QName(internal.State.ValueAsQName(isafestate.TValue(v), name))
}

func (v TValue) AsBool(name string) bool {
	return internal.State.ValueAsBool(isafestate.TValue(v), name)
}

func (v TValue) AsValue(name string) TValue {
	return TValue(internal.State.ValueAsValue(isafestate.TValue(v), name))
}

func (v TValue) Len() int {
	return internal.State.ValueLen(isafestate.TValue(v))
}

func (v TValue) GetAsValue(index int) TValue {
	return TValue(internal.State.ValueGetAsValue(isafestate.TValue(v), index))
}

func (v TValue) GetAsInt32(index int) int32 {
	return internal.State.ValueGetAsInt32(isafestate.TValue(v), index)
}

func (v TValue) GetAsInt64(index int) int64 {
	return internal.State.ValueGetAsInt64(isafestate.TValue(v), index)
}

func (v TValue) GetAsFloat32(index int) float32 {
	return internal.State.ValueGetAsFloat32(isafestate.TValue(v), index)
}

func (v TValue) GetAsFloat64(index int) float64 {
	return internal.State.ValueGetAsFloat64(isafestate.TValue(v), index)
}

func (v TValue) GetAsBytes(index int) []byte {
	return internal.State.ValueGetAsBytes(isafestate.TValue(v), index)
}

func (v TValue) GetAsQName(index int) QName {
	return QName(internal.State.ValueGetAsQName(isafestate.TValue(v), index))
}

func (v TValue) GetAsBool(index int) bool {
	return internal.State.ValueGetAsBool(isafestate.TValue(v), index)
}

func (v TValue) GetAsString(index int) string {
	return internal.State.ValueGetAsString(isafestate.TValue(v), index)
}
