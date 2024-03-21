/*
* Copyright (c) 2023-present unTill Pro, Ltd.
*  @author Michael Saigachenko
 */

package exttinygo

import (
	"unsafe"
)

func (v TValue) Len() int {
	return int(hostValueLength(uint64(v)))
}

func (v TValue) AsString(name string) string {
	ptr := hostValueAsString(uint64(v), uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name)))
	return decodeString(ptr)
}

func (v TValue) AsBytes(name string) (ret []byte) {
	ptr := hostValueAsBytes(uint64(v), uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name)))
	return decodeSlice(ptr)
}

func (v TValue) AsInt32(name string) int32 {
	return int32(hostValueAsInt32(uint64(v), uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name))))
}

func (v TValue) AsInt64(name string) int64 {
	return int64(hostValueAsInt64(uint64(v), uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name))))
}

func (v TValue) AsFloat32(name string) float32 {
	return hostValueAsFloat32(uint64(v), uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name)))
}

func (v TValue) AsFloat64(name string) float64 {
	return hostValueAsFloat64(uint64(v), uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name)))
}

func (v TValue) AsQName(name string) QName {
	pkgPtr := hostValueAsQNamePkg(uint64(v), uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name)))
	entityPtr := hostValueAsQNameEntity(uint64(v), uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name)))
	return QName{
		Pkg:    decodeString(pkgPtr),
		Entity: decodeString(entityPtr),
	}
}

func (v TValue) AsBool(name string) bool {
	return hostValueAsBool(uint64(v), uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name))) > 0
}

func (v TValue) AsValue(name string) TValue {
	return TValue(hostValueAsValue(uint64(v), uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name))))
}

func (v TValue) GetAsBytes(index int) (ret []byte) {
	return decodeSlice(hostValueGetAsBytes(uint64(v), uint32(index)))
}

func (v TValue) GetAsString(index int) string {
	return decodeString(hostValueGetAsString(uint64(v), uint32(index)))
}

func (v TValue) GetAsInt32(index int) int32 {
	return int32(hostValueGetAsInt32(uint64(v), uint32(index)))
}

func (v TValue) GetAsInt64(index int) int64 {
	return int64(hostValueGetAsInt64(uint64(v), uint32(index)))
}

func (v TValue) GetAsFloat32(index int) float32 {
	return hostValueGetAsFloat32(uint64(v), uint32(index))
}

func (v TValue) GetAsFloat64(index int) float64 {
	return hostValueGetAsFloat64(uint64(v), uint32(index))
}

func (v TValue) GetAsValue(index int) TValue {
	return TValue(hostValueGetAsValue(uint64(v), uint32(index)))
}

func (v TValue) GetAsQName(index int) QName {
	pkgPtr := hostValueGetAsQNamePkg(uint64(v), uint32(index))
	entityPtr := hostValueGetAsQNameEntity(uint64(v), uint32(index))
	return QName{
		Pkg:    decodeString(pkgPtr),
		Entity: decodeString(entityPtr),
	}
}

func (v TValue) GetAsBool(index int) bool {
	return hostValueGetAsBool(uint64(v), uint32(index)) > 0
}

func decodeSlice(value uint64) []byte {
	u := uintptr(uint32(value >> 32))
	s := uint32(value)
	return unsafe.Slice((*byte)(unsafe.Pointer(u)), s)
}

func decodeString(value uint64) (ret string) {
	u := uintptr(uint32(value >> 32))
	s := uint32(value)
	return unsafe.String((*byte)(unsafe.Pointer(u)), s)
}
