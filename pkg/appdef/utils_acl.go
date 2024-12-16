/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"strings"
)

// Returns "grant" if policy is allow, "revoke" if deny
func (p PolicyKind) ActionString() string {
	switch p {
	case PolicyKind_Allow:
		return "GRANT"
	case PolicyKind_Deny:
		return "REVOKE"
	}
	return p.TrimString()
}

// Renders an PolicyKind in human-readable form, without "PolicyKind_" prefix,
// suitable for debugging or error messages
func (p PolicyKind) TrimString() string {
	const pref = "PolicyKind_"
	return strings.TrimPrefix(p.String(), pref)
}
