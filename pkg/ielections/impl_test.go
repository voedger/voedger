/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */
package ielections

import (
	"strconv"
	"testing"
)

func TestElections(t *testing.T) {
	ttlStorage := newMockStorage[string, string]()
	counter := 0
	ElectionsTestSuite(t, ttlStorage, TestDataGen[string, string]{
		NextKey: func() string {
			counter++
			return "testKey" + strconv.Itoa(counter)
		},
		NextVal: func() string {
			counter++
			return "testVal" + strconv.Itoa(counter)
		},
	})
}
