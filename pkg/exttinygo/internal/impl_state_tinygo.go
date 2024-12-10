/*
  - Copyright (c) 2023-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/

// nolint G115
package internal

import (
	"unsafe"

	safe "github.com/voedger/voedger/pkg/state/isafestateapi"
)

const maxUint = ^uint64(0)

type hostSafeStateAPI struct{}

func (hostSafeStateAPI) KeyBuilder(storage, entity string) safe.TKeyBuilder {
	return safe.TKeyBuilder(hostGetKey(uint32(uintptr(unsafe.Pointer(unsafe.StringData(storage)))), uint32(len(storage)),
		uint32(uintptr(unsafe.Pointer(unsafe.StringData(entity)))), uint32(len(entity))))
}

func (hostSafeStateAPI) MustGetValue(key safe.TKeyBuilder) safe.TValue {
	return safe.TValue(hostGetValue(uint64(key)))
}

func (hostSafeStateAPI) QueryValue(key safe.TKeyBuilder) (safe.TValue, bool) {
	id := hostQueryValue(uint64(key))
	if id != maxUint {
		return safe.TValue(id), true
	}
	return safe.TValue(0), false
}

func (hostSafeStateAPI) NewValue(key safe.TKeyBuilder) safe.TIntent {
	return safe.TIntent(hostNewValue(uint64(key)))
}

func (hostSafeStateAPI) UpdateValue(key safe.TKeyBuilder, existingValue safe.TValue) safe.TIntent {
	return safe.TIntent(hostUpdateValue(uint64(key), uint64(existingValue)))
}

var CurrentReadCallback func(key safe.TKey, value safe.TValue)

func (hostSafeStateAPI) ReadValues(key safe.TKeyBuilder, callback func(key safe.TKey, value safe.TValue)) {
	CurrentReadCallback = callback
	hostReadValues(uint64(key))
}

// Key Builder
func (hostSafeStateAPI) KeyBuilderPutInt32(key safe.TKeyBuilder, name string, value int32) {
	hostRowWriterPutInt32(uint64(key), 0, uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name)), uint32(value))
}

func (hostSafeStateAPI) KeyBuilderPutInt64(key safe.TKeyBuilder, name string, value int64) {
	hostRowWriterPutInt64(uint64(key), 0, uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name)), uint64(value))
}

func (hostSafeStateAPI) KeyBuilderPutRecordID(key safe.TKeyBuilder, name string, value int64) {
	hostRowWriterPutInt64(uint64(key), 0, uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name)), uint64(value))
}

func (hostSafeStateAPI) KeyBuilderPutFloat32(key safe.TKeyBuilder, name string, value float32) {
	hostRowWriterPutFloat32(uint64(key), 0, uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name)), value)
}

func (hostSafeStateAPI) KeyBuilderPutFloat64(key safe.TKeyBuilder, name string, value float64) {
	hostRowWriterPutFloat64(uint64(key), 0, uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name)), value)
}

func (hostSafeStateAPI) KeyBuilderPutString(key safe.TKeyBuilder, name string, value string) {
	hostRowWriterPutString(uint64(key), 0, uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name)), uint32(uintptr(unsafe.Pointer(unsafe.StringData(value)))), uint32(len(value)))
}

func (hostSafeStateAPI) KeyBuilderPutBytes(key safe.TKeyBuilder, name string, value []byte) {
	hostRowWriterPutBytes(uint64(key), 0, uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name)), uint32(uintptr(unsafe.Pointer(unsafe.SliceData(value)))), uint32(len(value)))
}

func (hostSafeStateAPI) KeyBuilderPutQName(key safe.TKeyBuilder, name string, value safe.QName) {
	hostRowWriterPutQName(uint64(key), 0, uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name)),
		uint32(uintptr(unsafe.Pointer(unsafe.StringData(value.FullPkgName)))), uint32(len(value.FullPkgName)),
		uint32(uintptr(unsafe.Pointer(unsafe.StringData(value.Entity)))), uint32(len(value.Entity)))
}

func (hostSafeStateAPI) KeyBuilderPutBool(key safe.TKeyBuilder, name string, value bool) {
	var v uint32
	if value {
		v = 1
	}
	hostRowWriterPutBool(uint64(key), 0, uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name)), v)
}

// Intent

func (hostSafeStateAPI) IntentPutInt64(i safe.TIntent, name string, value int64) {
	hostRowWriterPutInt64(uint64(i), 1, uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name)), uint64(value))
}

func (hostSafeStateAPI) IntentPutInt32(i safe.TIntent, name string, value int32) {
	hostRowWriterPutInt32(uint64(i), 1, uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name)), uint32(value))
}

func (hostSafeStateAPI) IntentPutFloat32(i safe.TIntent, name string, value float32) {
	hostRowWriterPutFloat32(uint64(i), 1, uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name)), value)
}

func (hostSafeStateAPI) IntentPutFloat64(i safe.TIntent, name string, value float64) {
	hostRowWriterPutFloat64(uint64(i), 1, uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name)), value)
}

func (hostSafeStateAPI) IntentPutString(i safe.TIntent, name string, value string) {
	hostRowWriterPutString(uint64(i), 1, uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name)), uint32(uintptr(unsafe.Pointer(unsafe.StringData(value)))), uint32(len(value)))
}

func (hostSafeStateAPI) IntentPutBytes(i safe.TIntent, name string, value []byte) {
	hostRowWriterPutBytes(uint64(i), 1, uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name)), uint32(uintptr(unsafe.Pointer(unsafe.SliceData(value)))), uint32(len(value)))
}

func (hostSafeStateAPI) IntentPutQName(i safe.TIntent, name string, value safe.QName) {
	hostRowWriterPutQName(uint64(i), 1,
		uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name)),
		uint32(uintptr(unsafe.Pointer(unsafe.StringData(value.FullPkgName)))), uint32(len(value.FullPkgName)),
		uint32(uintptr(unsafe.Pointer(unsafe.StringData(value.Entity)))), uint32(len(value.Entity)),
	)
}

func (hostSafeStateAPI) IntentPutBool(i safe.TIntent, name string, value bool) {
	var v uint32
	if value {
		v = 1
	}
	hostRowWriterPutBool(uint64(i), 1, uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name)), v)
}

// Value
func (hostSafeStateAPI) ValueAsValue(v safe.TValue, name string) safe.TValue {
	return safe.TValue(hostValueAsValue(uint64(v), uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name))))
}

func (hostSafeStateAPI) ValueAsInt32(v safe.TValue, name string) int32 {
	return int32(hostValueAsInt32(uint64(v), uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name))))
}

func (hostSafeStateAPI) ValueAsInt64(v safe.TValue, name string) int64 {
	return int64(hostValueAsInt64(uint64(v), uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name))))
}

func (hostSafeStateAPI) ValueAsFloat32(v safe.TValue, name string) float32 {
	return hostValueAsFloat32(uint64(v), uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name)))
}

func (hostSafeStateAPI) ValueAsFloat64(v safe.TValue, name string) float64 {
	return hostValueAsFloat64(uint64(v), uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name)))
}

func (hostSafeStateAPI) ValueAsString(v safe.TValue, name string) string {
	ptr := hostValueAsString(uint64(v), uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name)))
	return decodeString(ptr)
}

func (hostSafeStateAPI) ValueAsBytes(v safe.TValue, name string) []byte {
	ptr := hostValueAsBytes(uint64(v), uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name)))
	return decodeSlice(ptr)
}

func (hostSafeStateAPI) ValueAsQName(v safe.TValue, name string) safe.QName {
	pkgPtr := hostValueAsQNamePkg(uint64(v), uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name)))
	entityPtr := hostValueAsQNameEntity(uint64(v), uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name)))
	return safe.QName{
		FullPkgName: decodeString(pkgPtr),
		Entity:      decodeString(entityPtr),
	}
}

func (hostSafeStateAPI) ValueAsBool(v safe.TValue, name string) bool {
	return hostValueAsBool(uint64(v), uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name))) > 0
}

func (hostSafeStateAPI) ValueGetAsBytes(v safe.TValue, index int) []byte {
	ptr := hostValueGetAsBytes(uint64(v), uint32(index))
	return decodeSlice(ptr)
}

func (hostSafeStateAPI) ValueGetAsString(v safe.TValue, index int) string {
	ptr := hostValueGetAsString(uint64(v), uint32(index))
	return decodeString(ptr)
}

func (hostSafeStateAPI) ValueGetAsQName(v safe.TValue, index int) safe.QName {
	pkgPtr := hostValueGetAsQNamePkg(uint64(v), uint32(index))
	entityPtr := hostValueGetAsQNameEntity(uint64(v), uint32(index))
	return safe.QName{
		FullPkgName: decodeString(pkgPtr),
		Entity:      decodeString(entityPtr),
	}
}

func (hostSafeStateAPI) ValueGetAsBool(v safe.TValue, index int) bool {
	return hostValueGetAsBool(uint64(v), uint32(index)) > 0
}

func (hostSafeStateAPI) ValueGetAsInt32(v safe.TValue, index int) int32 {
	return int32(hostValueGetAsInt32(uint64(v), uint32(index)))
}

func (hostSafeStateAPI) ValueGetAsInt64(v safe.TValue, index int) int64 {
	return int64(hostValueGetAsInt64(uint64(v), uint32(index)))
}

func (hostSafeStateAPI) ValueGetAsFloat32(v safe.TValue, index int) float32 {
	return hostValueGetAsFloat32(uint64(v), uint32(index))
}

func (hostSafeStateAPI) ValueGetAsFloat64(v safe.TValue, index int) float64 {
	return hostValueGetAsFloat64(uint64(v), uint32(index))
}

func (hostSafeStateAPI) ValueLen(v safe.TValue) int {
	return int(hostValueLength(uint64(v)))
}

func (hostSafeStateAPI) ValueGetAsValue(v safe.TValue, index int) safe.TValue {
	return safe.TValue(hostValueGetAsValue(uint64(v), uint32(index)))
}

// Key
func (hostSafeStateAPI) KeyAsInt32(k safe.TKey, name string) int32 {
	return int32(hostKeyAsInt32(uint64(k), uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name))))
}

func (hostSafeStateAPI) KeyAsInt64(k safe.TKey, name string) int64 {
	return int64(hostKeyAsInt64(uint64(k), uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name))))
}

func (hostSafeStateAPI) KeyAsFloat32(k safe.TKey, name string) float32 {
	return hostKeyAsFloat32(uint64(k), uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name)))
}

func (hostSafeStateAPI) KeyAsFloat64(k safe.TKey, name string) float64 {
	return hostKeyAsFloat64(uint64(k), uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name)))
}

func (hostSafeStateAPI) KeyAsBytes(k safe.TKey, name string) []byte {
	return decodeSlice(hostKeyAsBytes(uint64(k), uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name))))
}

func (hostSafeStateAPI) KeyAsString(k safe.TKey, name string) string {
	return decodeString(hostKeyAsString(uint64(k), uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name))))
}

func (hostSafeStateAPI) KeyAsQName(k safe.TKey, name string) safe.QName {
	pkgPtr := hostKeyAsQNamePkg(uint64(k), uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name)))
	entityPtr := hostKeyAsQNameEntity(uint64(k), uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name)))
	return safe.QName{
		FullPkgName: decodeString(pkgPtr),
		Entity:      decodeString(entityPtr),
	}
}

func (hostSafeStateAPI) KeyAsBool(k safe.TKey, name string) bool {
	return hostKeyAsBool(uint64(k), uint32(uintptr(unsafe.Pointer(unsafe.StringData(name)))), uint32(len(name))) > 0
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
