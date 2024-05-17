//go:build !tinygo

/*
* Copyright (c) 2023-present unTill Pro, Ltd.
*  @author Michael Saigachenko
 */

package internal

func hostRowWriterPutString(id uint64, typ uint32, namePtr, nameSize, valuePtr, valueSize uint32) {
}

func hostRowWriterPutBytes(id uint64, typ uint32, namePtr, nameSize, valuePtr, valueSize uint32) {
}

func hostRowWriterPutQName(id uint64, typ uint32, namePtr, nameSize, pkgPtr, pkgSize, entityPtr, entitySize uint32) {
}

func hostRowWriterPutBool(id uint64, typ uint32, namePtr, nameSize, value uint32) {
}

func hostRowWriterPutInt32(id uint64, typ uint32, namePtr, nameSize, value uint32) {
}

func hostRowWriterPutInt64(id uint64, typ uint32, namePtr, nameSize uint32, value uint64) {
}

func hostRowWriterPutFloat32(id uint64, typ uint32, namePtr, nameSize uint32, value float32) {
}

func hostRowWriterPutFloat64(id uint64, typ uint32, namePtr, nameSize uint32, value float64) {
}

func hostGetKey(storagePtr, storageSize, entityPtr, entitySize uint32) uint64 {
	return 0
}

func hostQueryValue(keyId uint64) (result uint64) {
	return 0
}

func hostNewValue(keyId uint64) uint64 {
	return 0
}

func hostUpdateValue(keyId uint64, existingValueId uint64) uint64 {
	return 0
}

func hostValueLength(id uint64) uint32 {
	return 0
}

func hostValueAsBytes(id uint64, namePtr, nameSize uint32) uint64 {
	return 0
}

func hostValueAsString(id uint64, namePtr, nameSize uint32) uint64 {
	return 0
}

func hostValueAsInt32(id uint64, namePtr, nameSize uint32) uint32 {
	return 0
}

func hostValueAsInt64(id uint64, namePtr, nameSize uint32) uint64 {
	return 0
}

func hostValueAsFloat32(id uint64, namePtr, nameSize uint32) float32 {
	return 0
}

func hostValueAsFloat64(id uint64, namePtr, nameSize uint32) float64 {
	return 0
}

func hostValueAsValue(id uint64, namePtr, nameSize uint32) uint64 {
	return 0
}

func hostValueAsQNamePkg(id uint64, namePtr, nameSize uint32) uint64 {
	return 0
}

func hostValueAsQNameEntity(id uint64, namePtr, nameSize uint32) uint64 {
	return 0
}

func hostValueAsBool(id uint64, namePtr, nameSize uint32) uint64 {
	return 0
}

func hostValueGetAsBytes(id uint64, index uint32) uint64 {
	return 0
}

func hostValueGetAsString(id uint64, index uint32) uint64 {
	return 0
}

func hostValueGetAsInt32(id uint64, index uint32) uint32 {
	return 0
}

func hostValueGetAsInt64(id uint64, index uint32) uint64 {
	return 0
}

func hostValueGetAsFloat32(id uint64, index uint32) float32 {
	return 0
}

func hostValueGetAsFloat64(id uint64, index uint32) float64 {
	return 0
}

func hostValueGetAsValue(id uint64, index uint32) uint64 {
	return 0
}

func hostValueGetAsQNamePkg(id uint64, index uint32) uint64 {
	return 0
}

func hostValueGetAsQNameEntity(id uint64, index uint32) uint64 {
	return 0
}

func hostValueGetAsBool(id uint64, index uint32) uint64 {
	return 0
}

func hostKeyAsString(id uint64, namePtr, nameSize uint32) uint64 {
	return 0
}

func hostKeyAsBytes(id uint64, namePtr, nameSize uint32) uint64 {
	return 0
}

func hostKeyAsQNamePkg(id uint64, namePtr, nameSize uint32) uint64 {
	return 0
}

func hostKeyAsQNameEntity(id uint64, namePtr, nameSize uint32) uint64 {
	return 0
}

func hostKeyAsBool(id uint64, namePtr, nameSize uint32) uint64 {
	return 0
}

func hostKeyAsInt32(id uint64, namePtr, nameSize uint32) uint32 {
	return 0
}

func hostKeyAsInt64(id uint64, namePtr, nameSize uint32) uint64 {
	return 0
}

func hostKeyAsFloat32(id uint64, namePtr, nameSize uint32) float32 {
	return 0
}

func hostKeyAsFloat64(id uint64, namePtr, nameSize uint32) float64 {
	return 0
}

func hostReadValues(keyId uint64) {
}

func hostGetValue(keyId uint64) (result uint64) {
	return 0
}
