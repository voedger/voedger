/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts

import (
	"strings"

	"github.com/voedger/voedger/pkg/utils/utils"
)

func (k ProcessorKind) MarshalText() ([]byte, error) {
	var s string
	if k < ProcessorKind_Count {
		s = k.String()
	} else {
		s = utils.UintToString(k)
	}
	return []byte(s), nil
}

// Renders an ProcessorKind in human-readable form, without `ProcessorKind_` prefix,
// suitable for debugging or error messages
func (k ProcessorKind) TrimString() string {
	const pref = "ProcessorKind_"
	return strings.TrimPrefix(k.String(), pref)
}
