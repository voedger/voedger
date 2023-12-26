/*
* Copyright (c) 2021-present unTill Pro, Ltd.
*  @author Michael Saigachenko
 */

package extensions

import (
	"unsafe"
)

func (i TIntent) PutInt32(name string, value int32) {
	hostRowWriterPutInt32(uint64(i), 1, uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name)), uint32(value))
}

func (i TIntent) PutInt64(name string, value int64) {
	hostRowWriterPutInt64(uint64(i), 1, uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name)), uint64(value))
}

func (i TIntent) PutFloat32(name string, value float32) {
	hostRowWriterPutFloat32(uint64(i), 1, uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name)), value)
}

func (i TIntent) PutFloat64(name string, value float64) {
	hostRowWriterPutFloat64(uint64(i), 1, uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name)), value)
}

func (i TIntent) PutString(name string, value string) {
	hostRowWriterPutString(uint64(i), 1, uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name)), uint32(uintptr(unsafe.Pointer(unsafe.StringData(value)))), uint32(len(value)))
}

func (i TIntent) PutBytes(name string, value []byte) {
	hostRowWriterPutBytes(uint64(i), 1, uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name)), uint32(uintptr(unsafe.Pointer(unsafe.SliceData(value)))), uint32(len(value)))
}

func (i TIntent) PutQName(name string, value QName) {
	hostRowWriterPutQName(uint64(i), 1,
		uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name)),
		uint32(uintptr(unsafe.Pointer(unsafe.StringData(value.Pkg)))), uint32(len(value.Pkg)),
		uint32(uintptr(unsafe.Pointer(unsafe.StringData(value.Entity)))), uint32(len(value.Entity)),
	)
}

func (i TIntent) PutBool(name string, value bool) {
	var v uint32
	if value {
		v = 1
	}
	hostRowWriterPutBool(uint64(i), 1, uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name)), v)
}
