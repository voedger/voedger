/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"strings"

	"github.com/voedger/voedger/pkg/coreutils/utils"
)

func (k FilterKind) MarshalText() ([]byte, error) {
	var s string
	if k < FilterKind_count {
		s = k.String()
	} else {
		s = utils.UintToString(k)
	}
	return []byte(s), nil
}

// Renders an FilterKind in human-readable form, without "FilterKind_" prefix,
// suitable for debugging or error messages
func (k FilterKind) TrimString() string {
	const pref = "FilterKind_"
	return strings.TrimPrefix(k.String(), pref)
}
