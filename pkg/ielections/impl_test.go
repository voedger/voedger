/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */
package ielections

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// [~server.design.orch/ElectionsTest~impl]
func TestElections(t *testing.T) {
	ttlStorage := newTTLStorageMock[string, string]()
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

func TestDurationMult(t *testing.T) {
	expectedDuration, err := time.ParseDuration("7.5s")
	require.NoError(t, err)
	actualDuration := durationMult(10, 0.75)
	require.Equal(t, expectedDuration, actualDuration)
}
