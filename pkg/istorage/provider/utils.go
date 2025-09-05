/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package provider

import "github.com/google/uuid"

func NewTestKeyspaceIsolationSuffix() string {
	res := uuid.NewString()
	return res[:8]
}
