//go:build !tinygo

/*
* Copyright (c) 2023-present unTill Pro, Ltd.
*  @author Michael Saigachenko
 */

package internal

func hostRowWriterPutString(_ uint64, _ uint32, _, _, _, _ uint32) {
}

func hostRowWriterPutBytes(_ uint64, _ uint32, _, _, _, _ uint32) {
}

func hostRowWriterPutQName(_ uint64, _ uint32, _, _, _, _, _, _ uint32) {
}

func hostRowWriterPutBool(_ uint64, _ uint32, _, _, _ uint32) {
}

func hostRowWriterPutInt32(_ uint64, _ uint32, _, _, _ uint32) {
}

func hostRowWriterPutInt64(_ uint64, _ uint32, _, _ uint32, _ uint64) {
}

func hostRowWriterPutFloat32(_ uint64, _ uint32, _, _ uint32, _ float32) {
}

func hostRowWriterPutFloat64(_ uint64, _ uint32, _, _ uint32, _ float64) {
}

func hostGetKey(_, _, _, _ uint32) uint64 {
	return 0
}

func hostQueryValue(_ uint64) (_ uint64) {
	return 0
}

func hostNewValue(_ uint64) uint64 {
	return 0
}

func hostUpdateValue(_ uint64, _ uint64) uint64 {
	return 0
}

func hostValueLength(_ uint64) uint32 {
	return 0
}

func hostValueAsBytes(_ uint64, _, _ uint32) uint64 {
	return 0
}

func hostValueAsString(_ uint64, _, _ uint32) uint64 {
	return 0
}

func hostValueAsInt32(_ uint64, _, _ uint32) uint32 {
	return 0
}

func hostValueAsInt64(_ uint64, _, _ uint32) uint64 {
	return 0
}

func hostValueAsFloat32(_ uint64, _, _ uint32) float32 {
	return 0
}

func hostValueAsFloat64(_ uint64, _, _ uint32) float64 {
	return 0
}

func hostValueAsValue(_ uint64, _, _ uint32) uint64 {
	return 0
}

func hostValueAsQNamePkg(_ uint64, _, _ uint32) uint64 {
	return 0
}

func hostValueAsQNameEntity(_ uint64, _, _ uint32) uint64 {
	return 0
}

func hostValueAsBool(_ uint64, _, _ uint32) uint64 {
	return 0
}

func hostValueGetAsBytes(_ uint64, _ uint32) uint64 {
	return 0
}

func hostValueGetAsString(_ uint64, _ uint32) uint64 {
	return 0
}

func hostValueGetAsInt32(_ uint64, _ uint32) uint32 {
	return 0
}

func hostValueGetAsInt64(_ uint64, _ uint32) uint64 {
	return 0
}

func hostValueGetAsFloat32(_ uint64, _ uint32) float32 {
	return 0
}

func hostValueGetAsFloat64(_ uint64, _ uint32) float64 {
	return 0
}

func hostValueGetAsValue(_ uint64, _ uint32) uint64 {
	return 0
}

func hostValueGetAsQNamePkg(_ uint64, _ uint32) uint64 {
	return 0
}

func hostValueGetAsQNameEntity(_ uint64, _ uint32) uint64 {
	return 0
}

func hostValueGetAsBool(_ uint64, _ uint32) uint64 {
	return 0
}

func hostKeyAsString(_ uint64, _, _ uint32) uint64 {
	return 0
}

func hostKeyAsBytes(_ uint64, _, _ uint32) uint64 {
	return 0
}

func hostKeyAsQNamePkg(_ uint64, _, _ uint32) uint64 {
	return 0
}

func hostKeyAsQNameEntity(_ uint64, _, _ uint32) uint64 {
	return 0
}

func hostKeyAsBool(_ uint64, _, _ uint32) uint64 {
	return 0
}

func hostKeyAsInt32(_ uint64, _, _ uint32) uint32 {
	return 0
}

func hostKeyAsInt64(_ uint64, _, _ uint32) uint64 {
	return 0
}

func hostKeyAsFloat32(_ uint64, _, _ uint32) float32 {
	return 0
}

func hostKeyAsFloat64(_ uint64, _, _ uint32) float64 {
	return 0
}

func hostReadValues(_ uint64) {
}

func hostGetValue(_ uint64) (_ uint64) {
	return 0
}
