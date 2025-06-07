/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"strings"

	"github.com/voedger/voedger/pkg/coreutils/utils"
)

func (k ExtensionEngineKind) MarshalText() ([]byte, error) {
	var s string
	if k < ExtensionEngineKind_count {
		s = k.String()
	} else {
		s = utils.UintToString(k)
	}
	return []byte(s), nil
}

// Renders an ExtensionEngineKind in human-readable form, without "ExtensionEngineKind_" prefix,
// suitable for debugging or error messages
func (k ExtensionEngineKind) TrimString() string {
	const pref = "ExtensionEngineKind_"
	return strings.TrimPrefix(k.String(), pref)
}
