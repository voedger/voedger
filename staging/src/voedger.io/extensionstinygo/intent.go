/*
* Copyright (c) 2021-present unTill Pro, Ltd.
*  @author Michael Saigachenko
 */

package extensions

import (
	"reflect"
	"unsafe"
)

func (i TIntent) PutInt32(name string, value int32) {
	nh := (*reflect.StringHeader)(unsafe.Pointer(&name))
	hostRowWriterPutInt32(uint64(i), 1, uint32(nh.Data), uint32(nh.Len), uint32(value))
}

func (i TIntent) PutInt64(name string, value int64) {
	nh := (*reflect.StringHeader)(unsafe.Pointer(&name))
	hostRowWriterPutInt64(uint64(i), 1, uint32(nh.Data), uint32(nh.Len), uint64(value))
}

func (i TIntent) PutFloat32(name string, value float32) {
	nh := (*reflect.StringHeader)(unsafe.Pointer(&name))
	hostRowWriterPutFloat32(uint64(i), 1, uint32(nh.Data), uint32(nh.Len), value)
}

func (i TIntent) PutFloat64(name string, value float64) {
	nh := (*reflect.StringHeader)(unsafe.Pointer(&name))
	hostRowWriterPutFloat64(uint64(i), 1, uint32(nh.Data), uint32(nh.Len), value)
}

func (i TIntent) PutString(name string, value string) {
	nh := (*reflect.StringHeader)(unsafe.Pointer(&name))
	vh := (*reflect.StringHeader)(unsafe.Pointer(&value))
	hostRowWriterPutString(uint64(i), 1, uint32(nh.Data), uint32(nh.Len), uint32(vh.Data), uint32(vh.Len))
}

func (i TIntent) PutBytes(name string, value []byte) {
	nh := (*reflect.StringHeader)(unsafe.Pointer(&name))
	vh := (*reflect.SliceHeader)(unsafe.Pointer(&value))
	hostRowWriterPutBytes(uint64(i), 1, uint32(nh.Data), uint32(nh.Len), uint32(vh.Data), uint32(vh.Len))
}

func (i TIntent) PutQName(name string, value QName) {
	nh := (*reflect.StringHeader)(unsafe.Pointer(&name))
	pkg := value.Pkg
	entity := value.Entity
	pkgh := (*reflect.StringHeader)(unsafe.Pointer(&pkg))
	eh := (*reflect.StringHeader)(unsafe.Pointer(&entity))
	hostRowWriterPutQName(uint64(i), 1, uint32(nh.Data), uint32(nh.Len), uint32(pkgh.Data), uint32(pkgh.Len), uint32(eh.Data), uint32(eh.Len))
}

func (i TIntent) PutBool(name string, value bool) {
	var v uint32
	if value {
		v = 1
	}
	nh := (*reflect.StringHeader)(unsafe.Pointer(&name))
	hostRowWriterPutBool(uint64(i), 1, uint32(nh.Data), uint32(nh.Len), v)
}
