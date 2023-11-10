/*
* Copyright (c) 2023-present unTill Pro, Ltd.
*  @author Michael Saigachenko
 */

package extensions

import (
	"reflect"
	"unsafe"
)

func (v TValue) Length() uint32 {
	return hostValueLength(uint64(v))
}

func (v TValue) AsString(name string) string {
	nh := (*reflect.StringHeader)(unsafe.Pointer(&name))
	ptr := hostValueAsString(uint64(v), uint32(nh.Data), uint32(nh.Len))
	return decodeString(ptr)
}

func (v TValue) AsBytes(name string) (ret []byte) {
	nh := (*reflect.StringHeader)(unsafe.Pointer(&name))
	ptr := hostValueAsBytes(uint64(v), uint32(nh.Data), uint32(nh.Len))

	strHdr := (*reflect.SliceHeader)(unsafe.Pointer(&ret))
	strHdr.Data = uintptr(uint32(ptr >> 32))
	strHdr.Len = extint(uint32(ptr))
	return
}

func (v TValue) AsInt32(name string) int32 {
	nh := (*reflect.StringHeader)(unsafe.Pointer(&name))
	return int32(hostValueAsInt32(uint64(v), uint32(nh.Data), uint32(nh.Len)))
}

func (v TValue) AsInt64(name string) int64 {
	nh := (*reflect.StringHeader)(unsafe.Pointer(&name))
	return int64(hostValueAsInt64(uint64(v), uint32(nh.Data), uint32(nh.Len)))
}

func (v TValue) AsFloat32(name string) float32 {
	nh := (*reflect.StringHeader)(unsafe.Pointer(&name))
	return hostValueAsFloat32(uint64(v), uint32(nh.Data), uint32(nh.Len))
}

func (v TValue) AsFloat64(name string) float64 {
	nh := (*reflect.StringHeader)(unsafe.Pointer(&name))
	return hostValueAsFloat64(uint64(v), uint32(nh.Data), uint32(nh.Len))
}

func (v TValue) AsQName(name string) QName {
	nh := (*reflect.StringHeader)(unsafe.Pointer(&name))
	pkgPtr := hostValueAsQNamePkg(uint64(v), uint32(nh.Data), uint32(nh.Len))
	entityPtr := hostValueAsQNameEntity(uint64(v), uint32(nh.Data), uint32(nh.Len))
	return QName{
		Pkg:    decodeString(pkgPtr),
		Entity: decodeString(entityPtr),
	}
}

func (v TValue) AsBool(name string) bool {
	nh := (*reflect.StringHeader)(unsafe.Pointer(&name))
	return hostValueAsBool(uint64(v), uint32(nh.Data), uint32(nh.Len)) > 0
}

func (v TValue) AsValue(name string) TValue {
	nh := (*reflect.StringHeader)(unsafe.Pointer(&name))
	return TValue(hostValueAsValue(uint64(v), uint32(nh.Data), uint32(nh.Len)))
}

func (v TValue) GetAsBytes(index int) (ret []byte) {
	ptr := hostValueGetAsBytes(uint64(v), uint32(index))
	strHdr := (*reflect.SliceHeader)(unsafe.Pointer(&ret))
	strHdr.Data = uintptr(uint32(ptr >> 32))
	strHdr.Len = extint(uint32(ptr))
	return
}

func (v TValue) GetAsString(index int) string {
	ptr := hostValueGetAsString(uint64(v), uint32(index))
	return decodeString(ptr)
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

func decodeString(value uint64) (ret string) {
	strHdr := (*reflect.StringHeader)(unsafe.Pointer(&ret))
	strHdr.Data = uintptr(uint32(value >> 32))
	strHdr.Len = extint(uint32(value))
	return
}

//export hostValueLength
func hostValueLength(id uint64) uint32

//export hostValueAsBytes
func hostValueAsBytes(id uint64, namePtr, nameSize uint32) uint64

//export hostValueAsString
func hostValueAsString(id uint64, namePtr, nameSize uint32) uint64

//export hostValueAsInt32
func hostValueAsInt32(id uint64, namePtr, nameSize uint32) uint32

//export hostValueAsInt64
func hostValueAsInt64(id uint64, namePtr, nameSize uint32) uint64

//export hostValueAsFloat32
func hostValueAsFloat32(id uint64, namePtr, nameSize uint32) float32

//export hostValueAsFloat64
func hostValueAsFloat64(id uint64, namePtr, nameSize uint32) float64

//export hostValueAsValue
func hostValueAsValue(id uint64, namePtr, nameSize uint32) uint64

//export hostValueAsQNamePkg
func hostValueAsQNamePkg(id uint64, namePtr, nameSize uint32) uint64

//export hostValueAsQNameEntity
func hostValueAsQNameEntity(id uint64, namePtr, nameSize uint32) uint64

//export hostValueAsBool
func hostValueAsBool(id uint64, namePtr, nameSize uint32) uint64

//export hostValueGetAsBytes
func hostValueGetAsBytes(id uint64, index uint32) uint64

//export hostValueGetAsString
func hostValueGetAsString(id uint64, index uint32) uint64

//export hostValueGetAsInt32
func hostValueGetAsInt32(id uint64, index uint32) uint32

//export hostValueGetAsInt64
func hostValueGetAsInt64(id uint64, index uint32) uint64

//export hostValueGetAsFloat32
func hostValueGetAsFloat32(id uint64, index uint32) float32

//export hostValueGetAsFloat64
func hostValueGetAsFloat64(id uint64, index uint32) float64

//export hostValueGetAsValue
func hostValueGetAsValue(id uint64, index uint32) uint64

//export hostValueGetAsQNamePkg
func hostValueGetAsQNamePkg(id uint64, index uint32) uint64

//export hostValueGetAsQNameEntity
func hostValueGetAsQNameEntity(id uint64, index uint32) uint64

//export hostValueGetAsBool
func hostValueGetAsBool(id uint64, index uint32) uint64
