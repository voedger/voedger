/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package provider

import "github.com/google/uuid"

const testTeyspaceIsloationSuffixLen = 8

func NewTestKeyspaceIsolationSuffix() string {
	res := uuid.NewString()
	return res[:testTeyspaceIsloationSuffixLen]
}
