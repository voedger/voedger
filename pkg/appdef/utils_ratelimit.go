/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"strings"

	"github.com/voedger/voedger/pkg/coreutils/utils"
)

// Renders an RateScope in human-readable form, without `RateScope_` prefix,
// suitable for debugging or error messages
func (rs RateScope) TrimString() string {
	const pref = "RateScope" + "_"
	return strings.TrimPrefix(rs.String(), pref)
}

func (o LimitOption) MarshalText() ([]byte, error) {
	var s string
	if o < LimitOption_count {
		s = o.String()
	} else {
		s = utils.UintToString(o)
	}
	return []byte(s), nil
}

// Renders an LimitOption in human-readable form, without `LimitOption_` prefix,
// suitable for debugging or error messages
func (o LimitOption) TrimString() string {
	const pref = "LimitOption" + "_"
	return strings.TrimPrefix(o.String(), pref)
}
