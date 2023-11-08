/*
* Copyright (c) 2023-present unTill Pro, Ltd.
*  @author Michael Saigachenko
 */

package extensions

import (
	"reflect"
	"unsafe"
)

func keyBuilderImpl(storage, entity string) TKeyBuilder {
	sh := (*reflect.StringHeader)(unsafe.Pointer(&storage))
	eh := (*reflect.StringHeader)(unsafe.Pointer(&entity))
	return TKeyBuilder(hostGetKey(uint32(sh.Data), uint32(sh.Len), uint32(eh.Data), uint32(eh.Len)))
}

func (k TKeyBuilder) PutInt32(name string, value int32) {
	nh := (*reflect.StringHeader)(unsafe.Pointer(&name))
	hostRowWriterPutInt32(uint64(k), 0, uint32(nh.Data), uint32(nh.Len), uint32(value))
}

func (i TKeyBuilder) PutInt64(name string, value int64) {
	nh := (*reflect.StringHeader)(unsafe.Pointer(&name))
	hostRowWriterPutInt64(uint64(i), 0, uint32(nh.Data), uint32(nh.Len), uint64(value))
}

func (i TKeyBuilder) PutFloat32(name string, value float32) {
	nh := (*reflect.StringHeader)(unsafe.Pointer(&name))
	hostRowWriterPutFloat32(uint64(i), 0, uint32(nh.Data), uint32(nh.Len), value)
}

func (i TKeyBuilder) PutFloat64(name string, value float64) {
	nh := (*reflect.StringHeader)(unsafe.Pointer(&name))
	hostRowWriterPutFloat64(uint64(i), 0, uint32(nh.Data), uint32(nh.Len), value)
}

func (k TKeyBuilder) PutString(name string, value string) {
	nh := (*reflect.StringHeader)(unsafe.Pointer(&name))
	vh := (*reflect.StringHeader)(unsafe.Pointer(&value))
	hostRowWriterPutString(uint64(k), 0, uint32(nh.Data), uint32(nh.Len), uint32(vh.Data), uint32(vh.Len))
}

func (i TKeyBuilder) PutBytes(name string, value []byte) {
	nh := (*reflect.StringHeader)(unsafe.Pointer(&name))
	vh := (*reflect.SliceHeader)(unsafe.Pointer(&value))
	hostRowWriterPutBytes(uint64(i), 0, uint32(nh.Data), uint32(nh.Len), uint32(vh.Data), uint32(vh.Len))
}

func (i TKeyBuilder) PutQName(name string, value QName) {
	nh := (*reflect.StringHeader)(unsafe.Pointer(&name))
	pkg := value.Pkg
	entity := value.Entity
	pkgh := (*reflect.StringHeader)(unsafe.Pointer(&pkg))
	eh := (*reflect.StringHeader)(unsafe.Pointer(&entity))
	hostRowWriterPutQName(uint64(i), 0, uint32(nh.Data), uint32(nh.Len), uint32(pkgh.Data), uint32(pkgh.Len), uint32(eh.Data), uint32(eh.Len))
}

func (i TKeyBuilder) PutBool(name string, value bool) {
	var v uint32
	if value {
		v = 1
	}
	nh := (*reflect.StringHeader)(unsafe.Pointer(&name))
	hostRowWriterPutBool(uint64(i), 0, uint32(nh.Data), uint32(nh.Len), v)
}

//export hostGetKey
func hostGetKey(storagePtr, storageSize, entityPtr, entitySize uint32) uint64
