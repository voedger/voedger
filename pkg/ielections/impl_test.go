/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */
package ielections

import (
	"testing"
)

func TestElections(t *testing.T) {
	ttlStorage := newMockStorage[string, string]()
	ElectionsTestSuite(t, ttlStorage)
}
