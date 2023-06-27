/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"strconv"
	"strings"
)

//go:generate stringer -type=ExtensionEngineKind -output=extension-engine-kind_string.go

const (
	ExtensionEngineKind_null ExtensionEngineKind = iota
	ExtensionEngineKind_BuiltIn
	ExtensionEngineKind_WASM

	ExtensionEngineKind_FakeLast
)

func (k ExtensionEngineKind) MarshalText() ([]byte, error) {
	var s string
	if k < ExtensionEngineKind_FakeLast {
		s = k.String()
	} else {
		const base = 10
		s = strconv.FormatUint(uint64(k), base)
	}
	return []byte(s), nil
}

// Renders an ExtensionEngineKind in human-readable form, without "ExtensionEngineKind_" prefix,
// suitable for debugging or error messages
func (k ExtensionEngineKind) TrimString() string {
	const pref = "ExtensionEngineKind_"
	return strings.TrimPrefix(k.String(), pref)
}
