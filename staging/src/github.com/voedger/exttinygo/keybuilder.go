/*
* Copyright (c) 2023-present unTill Pro, Ltd.
*  @author Michael Saigachenko
 */

package extensions

import (
	"unsafe"
)

func keyBuilderImpl(storage, entity string) TKeyBuilder {
	return TKeyBuilder(hostGetKey(uint32(uintptr(unsafe.Pointer(unsafe.StringData(storage)))), uint32(len(storage)),
		uint32(uintptr(unsafe.Pointer(unsafe.StringData(entity)))), uint32(len(entity))))
}

func (k TKeyBuilder) PutInt32(name string, value int32) {
	hostRowWriterPutInt32(uint64(k), 0, uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name)), uint32(value))
}

func (i TKeyBuilder) PutInt64(name string, value int64) {
	hostRowWriterPutInt64(uint64(i), 0, uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name)), uint64(value))
}

func (i TKeyBuilder) PutFloat32(name string, value float32) {
	hostRowWriterPutFloat32(uint64(i), 0, uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name)), value)
}

func (i TKeyBuilder) PutFloat64(name string, value float64) {
	hostRowWriterPutFloat64(uint64(i), 0, uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name)), value)
}

func (k TKeyBuilder) PutString(name string, value string) {
	hostRowWriterPutString(uint64(k), 0, uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name)), uint32(uintptr(unsafe.Pointer(unsafe.StringData(value)))), uint32(len(value)))
}

func (i TKeyBuilder) PutBytes(name string, value []byte) {
	hostRowWriterPutBytes(uint64(i), 0, uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name)), uint32(uintptr(unsafe.Pointer(unsafe.SliceData(value)))), uint32(len(value)))
}

func (i TKeyBuilder) PutQName(name string, value QName) {
	hostRowWriterPutQName(uint64(i), 0, uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name)),
		uint32(uintptr(unsafe.Pointer(unsafe.StringData(value.Pkg)))), uint32(len(value.Entity)),
		uint32(uintptr(unsafe.Pointer(unsafe.StringData(value.Entity)))), uint32(len(value.Entity)),
	)
}

func (i TKeyBuilder) PutBool(name string, value bool) {
	var v uint32
	if value {
		v = 1
	}
	hostRowWriterPutBool(uint64(i), 0, uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name)), v)
}

//export hostGetKey
func hostGetKey(storagePtr, storageSize, entityPtr, entitySize uint32) uint64
