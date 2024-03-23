//go:build tinygo

/*
* Copyright (c) 2023-present unTill Pro, Ltd.
*  @author Michael Saigachenko
 */

package exttinygo

type extint = uintptr

//export hostPanic
func hostPanic(msgPtr, msgSize uint32)

//export hostRowWriterPutString
func hostRowWriterPutString(id uint64, typ uint32, namePtr, nameSize, valuePtr, valueSize uint32)

//export hostRowWriterPutBytes
func hostRowWriterPutBytes(id uint64, typ uint32, namePtr, nameSize, valuePtr, valueSize uint32)

//export hostRowWriterPutQName
func hostRowWriterPutQName(id uint64, typ uint32, namePtr, nameSize, pkgPtr, pkgSize, entityPtr, entitySize uint32)

//export hostRowWriterPutIntBool
func hostRowWriterPutBool(id uint64, typ uint32, namePtr, nameSize, value uint32)

//export hostRowWriterPutInt32
func hostRowWriterPutInt32(id uint64, typ uint32, namePtr, nameSize, value uint32)

//export hostRowWriterPutInt64
func hostRowWriterPutInt64(id uint64, typ uint32, namePtr, nameSize uint32, value uint64)

//export hostRowWriterPutFloat32
func hostRowWriterPutFloat32(id uint64, typ uint32, namePtr, nameSize uint32, value float32)

//export hostRowWriterPutFloat64
func hostRowWriterPutFloat64(id uint64, typ uint32, namePtr, nameSize uint32, value float64)

//export hostGetKey
func hostGetKey(storagePtr, storageSize, entityPtr, entitySize uint32) uint64

//export hostQueryValue
func hostQueryValue(keyId uint64) (result uint64)

//export hostNewValue
func hostNewValue(keyId uint64) uint64

//export hostUpdateValue
func hostUpdateValue(keyId uint64, existingValueId uint64) uint64

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

//export hostReadValues
func hostReadValues(keyId uint64)

//export hostGetValue
func hostGetValue(keyId uint64) (result uint64)
