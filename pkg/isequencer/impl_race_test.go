//go:build race
// +build race

/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package isequencer_test

import (
	"testing"
)

// lasts for ~140 seconds with -race
func TestLongRace(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	for range 1000 {
		TestISequencer_ComplexEvents(t)
		TestISequencer_Start(t)
		TestISequencer_Flush(t)
		TestISequencer_Next(t)
		TestISequencer_Actualize(t)
		TestISequencer_MultipleActualizes(t)
		TestISequencer_LongRecovery(t)
	}
}
