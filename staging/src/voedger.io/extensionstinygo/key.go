/*
* Copyright (c) 2023-present unTill Pro, Ltd.
*  @author Michael Saigachenko
 */

package extensions

import (
	"reflect"
	"unsafe"
)

type TKey uint64

func (v TKey) AsString(name string) string {
	nh := (*reflect.StringHeader)(unsafe.Pointer(&name))
	ptr := hostKeyAsString(uint64(v), uint32(nh.Data), uint32(nh.Len))
	return decodeString(ptr)
}

func (v TKey) AsInt32(name string) int32 {
	nh := (*reflect.StringHeader)(unsafe.Pointer(&name))
	return int32(hostKeyAsInt32(uint64(v), uint32(nh.Data), uint32(nh.Len)))
}

func (v TKey) AsInt64(name string) int64 {
	nh := (*reflect.StringHeader)(unsafe.Pointer(&name))
	return int64(hostKeyAsInt64(uint64(v), uint32(nh.Data), uint32(nh.Len)))
}

func (v TKey) AsFloat32(name string) float32 {
	nh := (*reflect.StringHeader)(unsafe.Pointer(&name))
	return float32(hostKeyAsFloat32(uint64(v), uint32(nh.Data), uint32(nh.Len)))
}

func (v TKey) AsFloat64(name string) float64 {
	nh := (*reflect.StringHeader)(unsafe.Pointer(&name))
	return float64(hostKeyAsFloat64(uint64(v), uint32(nh.Data), uint32(nh.Len)))
}

func (v TKey) AsBytes(name string) (ret []byte) {
	nh := (*reflect.StringHeader)(unsafe.Pointer(&name))
	ptr := hostKeyAsBytes(uint64(v), uint32(nh.Data), uint32(nh.Len))

	strHdr := (*reflect.SliceHeader)(unsafe.Pointer(&ret))
	strHdr.Data = uintptr(uint32(ptr >> 32))
	strHdr.Len = extint(uint32(ptr))
	return
}

func (v TKey) AsQName(name string) QName {
	nh := (*reflect.StringHeader)(unsafe.Pointer(&name))
	pkgPtr := hostKeyAsQNamePkg(uint64(v), uint32(nh.Data), uint32(nh.Len))
	entityPtr := hostKeyAsQNameEntity(uint64(v), uint32(nh.Data), uint32(nh.Len))
	return QName{
		Pkg:    decodeString(pkgPtr),
		Entity: decodeString(entityPtr),
	}
}

func (v TKey) AsBool(name string) bool {
	nh := (*reflect.StringHeader)(unsafe.Pointer(&name))
	ret := hostKeyAsBool(uint64(v), uint32(nh.Data), uint32(nh.Len))
	return ret > 0
}

//export hostKeyAsString
func hostKeyAsString(id uint64, namePtr, nameSize uint32) uint64

//export hostKeyAsBytes
func hostKeyAsBytes(id uint64, namePtr, nameSize uint32) uint64

//export hostKeyAsQNamePkg
func hostKeyAsQNamePkg(id uint64, namePtr, nameSize uint32) uint64

//export hostKeyAsQNameEntity
func hostKeyAsQNameEntity(id uint64, namePtr, nameSize uint32) uint64

//export hostKeyAsBool
func hostKeyAsBool(id uint64, namePtr, nameSize uint32) uint64

//export hostKeyAsInt32
func hostKeyAsInt32(id uint64, namePtr, nameSize uint32) uint32

//export hostKeyAsInt64
func hostKeyAsInt64(id uint64, namePtr, nameSize uint32) uint64

//export hostKeyAsFloat32
func hostKeyAsFloat32(id uint64, namePtr, nameSize uint32) float32

//export hostKeyAsFloat64
func hostKeyAsFloat64(id uint64, namePtr, nameSize uint32) float64
